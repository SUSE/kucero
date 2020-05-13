package node

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jenting/kucero/pkg/host"
)

type Worker struct {
	nodeName           string
	expiryTimeToRotate time.Duration
	clock              Clock
	certificates       map[string]string
}

// NewWorker returns a worker node certificate interface
func NewWorker(nodeName string, expiryTimeToRotate time.Duration) Certificate {
	certificates := make(map[string]string, 0)
	for k, v := range kubeletCertificates {
		certificates[k] = v
	}

	return &Worker{
		nodeName:           nodeName,
		expiryTimeToRotate: expiryTimeToRotate,
		clock:              NewRealClock(),
		certificates:       certificates,
	}
}

// CheckExpiration checks worker node certificate
// returns the certificates which are going to expires
func (w *Worker) CheckExpiration() (map[OWNER][]string, error) {
	expiryCertificates := map[OWNER][]string{}

	logrus.Infof("Commanding check %s node certificate expiration", w.nodeName)

	kubeletExpiryCertificates, err := kubeletCheckExpiration(w.expiryTimeToRotate, w.clock)
	if err != nil {
		return expiryCertificates, err
	}
	expiryCertificates["kubelet"] = kubeletExpiryCertificates

	return expiryCertificates, nil
}

// Rotate executes the steps to rotates the certificate
// including backing up certificate, rotates certificate, and restart kubelet
func (w *Worker) Rotate(expiryCertificates map[OWNER][]string) error {
	var errs error
	for owner, certificates := range expiryCertificates {
		for _, certificateName := range certificates {
			certificatePath, ok := w.certificates[certificateName]
			if !ok {
				continue
			}

			if err := backupCertificate(w.nodeName, certificateName, certificatePath); err != nil {
				errs = fmt.Errorf("%w; ", err)
				continue
			}

			if err := rotateCertificate(w.nodeName, owner, certificateName, certificatePath); err != nil {
				errs = fmt.Errorf("%w; ", err)
				continue
			}
		}
	}

	if err := host.RestartKubelet(w.nodeName); err != nil {
		errs = fmt.Errorf("%w; ", err)
	}

	return errs
}
