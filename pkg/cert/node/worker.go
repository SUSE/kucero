package node

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

var workerCertificates map[string]string

func init() {
	workerCertificates = make(map[string]string, 0)

	for k, v := range kubeletCertificates {
		workerCertificates[k] = v
	}
}

type Worker struct {
	nodeName           string
	expiryTimeToRotate time.Duration
	clock              Clock
}

// NewWorker returns a worker node certificate interface
func NewWorker(nodeName string, expiryTimeToRotate time.Duration) Certificate {
	return &Worker{
		nodeName:           nodeName,
		expiryTimeToRotate: expiryTimeToRotate,
		clock:              NewRealClock(),
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
	logrus.Infof("Commanding rotate %s node certificate", w.nodeName)

	var errs error
	for owner, certificates := range expiryCertificates {
		for _, certName := range certificates {
			_, ok := workerCertificates[certName]
			if !ok {
				continue
			}

			switch owner {
			case kubelet:
				if err := kubeletRenewCerts(certName); err != nil {
					errs = fmt.Errorf("%w; ", err)
					logrus.Errorf("Error invoking command: %v", err)
				}
			}
		}
	}

	return errs
}
