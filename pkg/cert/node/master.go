package node

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jenting/kucero/pkg/host"
	"github.com/sirupsen/logrus"
)

type Master struct {
	nodeName           string
	expiryTimeToRotate time.Duration
}

func NewMaster(nodeName string, expiryTimeToRotate time.Duration) Certificate {
	return &Master{
		nodeName:           nodeName,
		expiryTimeToRotate: expiryTimeToRotate,
	}
}

// CheckExpiration checks master node certificate
// returns true if one of the certificate
// is going to expiry
func (m *Master) CheckExpiration() ([]string, error) {
	expiryCertificates := []string{}

	logrus.Infof("Commanding check %s node certificate expiration", m.nodeName)

	// Relies on hostPID:true and privileged:true to enter host mount space
	cmd := host.NewCommandWithStdout("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/kubeadm", "alpha", "certs", "check-expiration")
	stdout, err := cmd.Output()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
		return expiryCertificates, err
	}

	stdoutS := string(stdout)
	kv := parseKubeadmCertsCheckExpiration(stdoutS)
	for cert, t := range kv {
		expiry := CheckExpiry(cert, t, m.expiryTimeToRotate)
		if expiry {
			expiryCertificates = append(expiryCertificates, cert)
		}
	}

	return expiryCertificates, nil
}

// Rotate executes the steps to rotates the certificate
// including backing up certificate, rotates certificate, and restart kubelet
func (m *Master) Rotate(expiryCertificates []string) error {
	if err := backupCertificate(m.nodeName, expiryCertificates); err != nil {
		logrus.Errorf("%v", err)
		return err
	}

	if err := rotateCertificate(m.nodeName, expiryCertificates); err != nil {
		logrus.Errorf("%v", err)
		return err
	}

	if err := host.RestartKubelet(m.nodeName); err != nil {
		logrus.Errorf("%v", err)
		return err
	}

	return nil
}

// backupCertificate backups the certificate/kubeconfig under folder /etc/kubernetes
// issued by kubeadm
func backupCertificate(nodeName string, expiryCertificates []string) error {
	logrus.Infof("Commanding backup %s node certs", nodeName)

	kubeadmCerts := map[string]string{
		"admin.conf":               "/etc/kubernetes/admin.conf",
		"controller-manager.conf":  "/etc/kubernetes/controller-manager.conf",
		"scheduler.conf":           "/etc/kubernetes/scheduler.conf",
		"apiserver":                "/etc/kubernetes/pki/apiserver.crt",
		"apiserver-etcd-client":    "/etc/kubernetes/pki/apiserver-etcd-client.crt",
		"apiserver-kubelet-client": "/etc/kubernetes/pki/apiserver-kubelet-client.crt",
		"front-proxy-client":       "/etc/kubernetes/pki/front-proxy-client.crt",
		"etcd-healthcheck-client":  "/etc/kubernetes/pki/etcd/healthcheck-client.crt",
		"etcd-peer":                "/etc/kubernetes/pki/etcd/peer.crt",
		"etcd-server":              "/etc/kubernetes/pki/etcd/server.crt",
	}

	var errors error
	for _, cert := range expiryCertificates {
		path, ok := kubeadmCerts[cert]
		if !ok {
			continue
		}

		dir := filepath.Dir(path)
		base := filepath.Base(path)
		ext := filepath.Ext(path)
		backupPath := filepath.Join(dir, strings.TrimSuffix(base, ext)+"-"+time.Now().Format("20060102030405")+ext+".bak")

		// Relies on hostPID:true and privileged:true to enter host mount space
		var err error
		cmd := host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/cp", path, backupPath)
		err = cmd.Run()
		if err != nil {
			errors = fmt.Errorf("%w; ", err)
			logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
		}
	}

	return errors
}

// rotateCertificate calls kubeadm alpha certs renew all
// on the host system to rotates kubeadm issued certificates
func rotateCertificate(nodeName string, expiryCerts []string) error {
	logrus.Infof("Commanding rotate %s node certificate", nodeName)

	// Relies on hostPID:true and privileged:true to enter host mount space
	cmd := host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/kubeadm", "alpha", "certs", "renew", "all")
	err := cmd.Run()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}

func parseKubeadmCertsCheckExpiration(input string) map[string]time.Time {
	certExpires := make(map[string]time.Time, 0)

	r := regexp.MustCompile("(.*) ([a-zA-Z]+ [0-9]{1,2}, [0-9]{4} [0-9]{1,2}:[0-9]{2} [a-zA-Z]+) (.*)")
	lines := strings.Split(input, "\n")
	parse := false
	for _, line := range lines {
		if parse {
			ss := r.FindStringSubmatch(line)
			if len(ss) < 3 {
				continue
			}

			cert := strings.TrimSpace(ss[1])
			t, err := time.Parse("Jan 02, 2006 15:04 MST", ss[2])
			if err != nil {
				fmt.Printf("err: %v\n", err)
				continue
			}

			certExpires[cert] = t
		}

		if strings.Contains(line, "CERTIFICATE") && strings.Contains(line, "EXPIRES") {
			parse = true
		}
	}

	return certExpires
}
