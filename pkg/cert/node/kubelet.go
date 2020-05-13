package node

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jenting/kucero/pkg/host"
)

var kubeletCertificates map[string]string = map[string]string{
	"kubelet": "/var/lib/kubelet/pki/kubelet.crt",
}

// kubeletCheckExpiration executes `openssl x509 -noout -enddate -in /var/lib/kubelet/pki/kubelet.crt`
// returns the certificates which are going to expires
func kubeletCheckExpiration(expiryTimeToRotate time.Duration, clock Clock) ([]string, error) {
	expiryCertificates := []string{}

	var errs error
	for name, path := range kubeletCertificates {
		// Relies on hostPID:true and privileged:true to enter host mount space
		cmd := host.NewCommandWithStdout("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/openssl", "x509", "-noout", "-enddate", "-in", path)
		stdout, err := cmd.Output()
		if err != nil {
			errs = fmt.Errorf("%w; ", err)
			logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
			continue
		}

		stdoutS := strings.TrimRight(string(stdout), "\n")
		t, err := parseOpensslEnddate(stdoutS)
		if err != nil {
			errs = fmt.Errorf("%w; ", err)
			continue
		}

		if checkCertificateExpiry(name, *t, expiryTimeToRotate, clock) {
			expiryCertificates = append(expiryCertificates, name)
		}
	}

	return expiryCertificates, errs
}

const (
	opensslConfTemplate = `
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[v3_req]
keyUsage = digitalSignature,keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[req_distinguished_name]
{{ range $value := .DN -}}
{{ $value }}
{{ end -}}

[alt_names]
{{ range $index, $value := .DNS -}}
DNS.{{ $index }} = {{ $value }}
{{ end -}}
{{ range $index, $value := .IP -}}
IP.{{ $index }} = {{ $value }}
{{ end -}}
`
)

func kubeletRenewCerts(certificateName, certificatePath string) error {
	// Relies on hostPID:true and privileged:true to enter host mount space

	// get distinguished name `openssl x509 -noout -subject -in <certificate-path>``
	cmd := host.NewCommandWithStdout("/usr/bin/nsenter", "-m/proc/1/ns/mnt",
		"/usr/bin/openssl", "x509", "-noout", "-subject", "-in", certificatePath)
	stdout, err := cmd.Output()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
		return err
	}

	stdoutS := strings.TrimRight(string(stdout), "\n")
	DN, err := parseOpensslSubjectDN(stdoutS)
	if err != nil {
		return err
	}

	logrus.Infof("Subject DN: %v\n", DN)

	// get x509v3 SAN `openssl x509 -noout -text -in <certificate-path> -certopt=no_subject,no_header,no_version,no_serial,no_signame,no_validity,no_issuer,no_pubkey,no_sigdump,no_aux`
	cmd = host.NewCommandWithStdout("/usr/bin/nsenter", "-m/proc/1/ns/mnt",
		"/usr/bin/openssl", "x509", "-noout", "-text", "-in", certificatePath,
		"-certopt=no_subject,no_header,no_version,no_serial,no_signame,no_validity,no_issuer,no_pubkey,no_sigdump,no_aux")
	stdout, err = cmd.Output()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
		return err
	}

	stdoutS = strings.TrimRight(string(stdout), "\n")
	DNS, IP := parseOpensslSubjectAltName(stdoutS)

	logrus.Infof("X509v3 SAN DNS: %v\n", DNS)
	logrus.Infof("X509v3 SAN IP: %v\n", IP)

	// generate openssl.conf
	template, err := template.New("").Parse(opensslConfTemplate)
	if err != nil {
		logrus.Errorf("Error new template: %v", err)
		return err
	}
	var rendered bytes.Buffer
	err = template.Execute(&rendered, struct {
		DN  []string
		DNS []string
		IP  []string
	}{
		DN:  DN,
		DNS: DNS,
		IP:  IP,
	})
	if err != nil {
		logrus.Errorf("Error render template: %v", err)
		return err
	}

	// TODO
	if err := ioutil.WriteFile("/tmp/openssl.conf", []byte(rendered.String()), 0600); err != nil {
		logrus.Errorf("Error write file: %v", err)
		return err
	}

	cmd = host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/dd", "of=/var/lib/kubelet/pki/openssl.conf", "<<<", fmt.Sprintf("%s", rendered.String()))
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
		return err
	}

	// generate CSR `openssl req -new -config=/var/lib/kubelet/pki/openssl.conf -key=/var/lib/kubelet/pki/kubelet.key out=/var/lib/kubelet/pki/kubelet.csr`
	cmd = host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt",
		"/usr/bin/openssl", "req", "-new", "-config=/var/lib/kubelet/pki/openssl.conf", "-key=/var/lib/kubelet/pki/kubelet.key", "-out=/var/lib/kubelet/pki/kubelet.csr")
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
		return err
	}

	// sign certificate with x509v3 SAN issuerd by kubelet CA
	// `openssl -req -days=365 -CA=/var/lib/kubelet/pki/kubelet-ca.crt -CAkey=/var/lib/kubelet/pki/kubelet-ca.key -in=/var/lib/kubelet/pki/kubelet.csr -out=/var/lib/kubelet/pki/kubelet.crt -sha256 -CAcreateserial -extfile=/var/lib/kubelet/pki/openssl.conf -extensions=v3_req
	cmd = host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt",
		"/usr/bin/openssl", "x509", "-req", "-days=365", "-CA=/var/lib/kubelet/pki/kubelet-ca.crt", "-CAkey=/var/lib/kubelet/pki/kubelet-ca.key",
		"-in=/var/lib/kubelet/pki/kubelet.csr", "-out=/var/lib/kubelet/pki/kubelet.crt",
		"-sha256", "-CAcreateserial", "-extfile=/var/lib/kubelet/pki/openssl.conf", "-extensions=v3_req")
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
		return err
	}

	return nil
}
