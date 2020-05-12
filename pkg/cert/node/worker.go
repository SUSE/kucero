package node

import (
	"errors"
	"strings"
	"time"

	"github.com/jenting/kucero/pkg/host"
	"github.com/sirupsen/logrus"
)

type Worker struct {
	nodeName           string
	expiryTimeToRotate time.Duration
}

func NewWorker(nodeName string, expiryTimeToRotate time.Duration) Certificate {
	return &Worker{
		nodeName:           nodeName,
		expiryTimeToRotate: expiryTimeToRotate,
	}
}

// CheckExpiration checks worker node certificate
func (w *Worker) CheckExpiration() ([]string, error) {
	expiryCertificates := []string{}
	certName := "kubelet"

	logrus.Infof("Commanding check %s node certificate expiration", w.nodeName)

	// Relies on hostPID:true and privileged:true to enter host mount space
	cmd := host.NewCommandWithStdout("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/openssl", "x509", "-noout", "-enddate", "-in", "/var/lib/kubelet/pki/kubelet.crt")
	stdout, err := cmd.Output()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
		return expiryCertificates, err
	}

	stdoutS := string(stdout)

	// notAfter=Jan 02 15:04:05 2006 MST
	if !strings.Contains(stdoutS, "notAfter") {
		logrus.Error("Cannot found notAfter key")
		return expiryCertificates, errors.New("Cannot found notAfter key")
	}

	ss := strings.Split(stdoutS, "=")
	if len(ss) < 2 {
		logrus.Error("Cannot found enddate")
		return expiryCertificates, errors.New("Cannot found endate")
	}
	ts := strings.TrimRight(ss[1], "\n")
	logrus.Infof("The certificate %s notAfter=%v", certName, ts)

	t, err := time.Parse("Jan 2 15:04:05 2006 MST", ts)
	if err != nil {
		logrus.Errorf("Error parse time: %v", err)
		return expiryCertificates, err
	}

	if CheckExpiry(certName, t, w.expiryTimeToRotate) {
		expiryCertificates = append(expiryCertificates, certName)
	}

	return expiryCertificates, nil
}

// Rotate executes the steps to rotates the certificate
// including backing up certificate, rotates certificate, and restart kubelet
func (w *Worker) Rotate(certs []string) error {
	// TODO: rotate worker node's kubelet server certificate
	return nil
}
