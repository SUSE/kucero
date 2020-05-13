package node

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jenting/kucero/pkg/host"
)

var kubeletCertificates map[string]string = map[string]string{
	"kubelet": "/var/lib/kubelet/pki/kubelet.crt",
}

// kubeletCheckExpiration executes `openssl x509 -noout -enddate -in /var/lib/kubelet/pki/kubelet.crt`
// returns the certificates which are going to expires
func kubeletCheckExpiration(expiryTimeToRotate time.Duration) ([]string, error) {
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

		stdoutS := string(stdout)
		t, err := parseKubeletCheckExpiration(stdoutS)
		if err != nil {
			errs = fmt.Errorf("%w; ", err)
			continue
		}

		if checkExpiry(name, *t, expiryTimeToRotate) {
			expiryCertificates = append(expiryCertificates, name)
		}
	}

	return expiryCertificates, errs
}

// parseKubeletCheckExpiration processes the `openssl x509 -noout -enddate -in <certificate-path>`
// output and returns the expires information
func parseKubeletCheckExpiration(input string) (*time.Time, error) {
	// notAfter=Jan 02 15:04:05 2006 MST
	if !strings.Contains(input, "notAfter") {
		err := errors.New("Cannot found notAfter key")
		logrus.Errorf("%v", err)
		return nil, err
	}

	ss := strings.Split(input, "=")
	if len(ss) < 2 {
		err := errors.New("Cannot found enddate")
		logrus.Errorf("%v", err)
		return nil, err
	}
	ts := strings.TrimRight(ss[1], "\n")

	t, err := time.Parse("Jan 2 15:04:05 2006 MST", ts)
	if err != nil {
		logrus.Errorf("%v", err)
		return nil, err
	}

	return &t, nil
}

func kubeletRenewCerts(certName string) error {
	// Relies on hostPID:true and privileged:true to enter host mount space
	// TODO
	return nil
}
