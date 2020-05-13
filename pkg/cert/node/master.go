package node

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jenting/kucero/pkg/host"
)

var masterCertificates map[string]string

func init() {
	masterCertificates = make(map[string]string, 0)

	for k, v := range kubeadmCertificates {
		masterCertificates[k] = v
	}
	for k, v := range kubeletCertificates {
		masterCertificates[k] = v
	}
}

type Master struct {
	nodeName           string
	expiryTimeToRotate time.Duration
}

// NewMaster returns a master node certificate interface
func NewMaster(nodeName string, expiryTimeToRotate time.Duration) Certificate {
	return &Master{
		nodeName:           nodeName,
		expiryTimeToRotate: expiryTimeToRotate,
	}
}

// CheckExpiration checks master node certificate
// returns the certificates which are going to expires
func (m *Master) CheckExpiration() (map[OWNER][]string, error) {
	expiryCertificates := map[OWNER][]string{}

	logrus.Infof("Commanding check %s node certificate expiration", m.nodeName)

	kubeadmExpiryCertificates, err := kubeadmCheckExpiration(m.expiryTimeToRotate)
	if err != nil {
		return expiryCertificates, err
	}
	expiryCertificates[kubeadm] = kubeadmExpiryCertificates

	kubeletExpiryCertificates, err := kubeletCheckExpiration(m.expiryTimeToRotate)
	if err != nil {
		return expiryCertificates, err
	}
	expiryCertificates[kubelet] = kubeletExpiryCertificates

	return expiryCertificates, nil
}

// Rotate executes the steps to rotates the certificate
// including backing up certificate, rotates certificate, and restart kubelet
func (m *Master) Rotate(expiryCertificates map[OWNER][]string) error {
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

// backupCertificate backups the certificate/kubeconfig
// under folder /etc/kubernetes issued by kubeadm
func backupCertificate(nodeName string, expiryCertificates map[OWNER][]string) error {
	logrus.Infof("Commanding backup %s node certs", nodeName)

	var errs error
	for _, certificates := range expiryCertificates {
		for _, certName := range certificates {
			path, ok := masterCertificates[certName]
			if !ok {
				continue
			}

			dir := filepath.Dir(path)
			base := filepath.Base(path)
			ext := filepath.Ext(path)
			backupPath := filepath.Join(dir, strings.TrimSuffix(base, ext)+"-"+time.Now().Format("20060102030405")+ext+".bak")

			// Relies on hostPID:true and privileged:true to enter host mount space
			cmd := host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/cp", path, backupPath)
			if err := cmd.Run(); err != nil {
				errs = fmt.Errorf("%w; ", err)
				logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
			}
		}
	}

	return errs
}

// rotateCertificate calls `kubeadm alpha certs renew <cert-name>`
// on the host system to rotates kubeadm issued certificates
func rotateCertificate(nodeName string, expiryCertificates map[OWNER][]string) error {
	logrus.Infof("Commanding rotate %s node certificate", nodeName)

	var errs error
	for owner, certificates := range expiryCertificates {
		for _, certName := range certificates {
			_, ok := masterCertificates[certName]
			if !ok {
				continue
			}

			switch owner {
			case kubeadm:
				if err := kubeadmRenewCerts(certName); err != nil {
					errs = fmt.Errorf("%w; ", err)
					logrus.Errorf("Error invoking command: %v", err)
				}
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
