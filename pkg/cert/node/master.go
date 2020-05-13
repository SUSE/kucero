package node

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jenting/kucero/pkg/host"
)

type Master struct {
	nodeName           string
	expiryTimeToRotate time.Duration
	clock              Clock
	certificates       map[string]string
}

// NewMaster returns a master node certificate interface
func NewMaster(nodeName string, expiryTimeToRotate time.Duration) Certificate {
	certificates := make(map[string]string, 0)
	for k, v := range kubeadmCertificates {
		certificates[k] = v
	}
	for k, v := range kubeletCertificates {
		certificates[k] = v
	}

	return &Master{
		nodeName:           nodeName,
		expiryTimeToRotate: expiryTimeToRotate,
		clock:              NewRealClock(),
		certificates:       certificates,
	}
}

// CheckExpiration checks master node certificate
// returns the certificates which are going to expires
func (m *Master) CheckExpiration() (map[OWNER][]string, error) {
	expiryCertificates := map[OWNER][]string{}

	logrus.Infof("Commanding check %s node certificate expiration", m.nodeName)

	kubeadmExpiryCertificates, err := kubeadmCheckExpiration(m.expiryTimeToRotate, m.clock)
	if err != nil {
		return expiryCertificates, err
	}
	expiryCertificates[kubeadm] = kubeadmExpiryCertificates

	kubeletExpiryCertificates, err := kubeletCheckExpiration(m.expiryTimeToRotate, m.clock)
	if err != nil {
		return expiryCertificates, err
	}
	expiryCertificates[kubelet] = kubeletExpiryCertificates

	return expiryCertificates, nil
}

// Rotate executes the steps to rotates the certificate
// including backing up certificate, rotates certificate, and restart kubelet
func (m *Master) Rotate(expiryCertificates map[OWNER][]string) error {
	var errs error
	for owner, certificates := range expiryCertificates {
		for _, certificateName := range certificates {
			certificatePath, ok := m.certificates[certificateName]
			if !ok {
				continue
			}

			if err := backupCertificate(m.nodeName, certificateName, certificatePath); err != nil {
				errs = fmt.Errorf("%w; ", err)
				continue
			}

			if err := rotateCertificate(m.nodeName, owner, certificateName, certificatePath); err != nil {
				errs = fmt.Errorf("%w; ", err)
				continue
			}
		}
	}

	if err := host.RestartKubelet(m.nodeName); err != nil {
		errs = fmt.Errorf("%w; ", err)
	}

	return errs
}
