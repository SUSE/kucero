package node

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenting/kucero/pkg/host"
	"github.com/sirupsen/logrus"
)

type Master struct {
	nodeName string
}

func NewMaster(nodeName string) Certificate {
	return &Master{
		nodeName: nodeName,
	}
}

func (m *Master) CheckExpiration() error {
	logrus.Infof("Commanding check %s node certificate expiration", m.nodeName)
	// Relies on hostPID:true and privileged:true to enter host mount space

	var err error
	cmd := host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/kubeadm", "alpha", "certs", "check-expiration")
	err = cmd.Run()
	if err != nil {
		logrus.Fatalf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}

// Rotate() executes the steps to rotates the certificate
// including backing up certificate, rotates certificate, and restart kubelet
func (m *Master) Rotate() error {
	if err := backupKubeconfig(m.nodeName); err != nil {
		logrus.Errorf("%v", err)
		return err
	}

	if err := backupCertificate(m.nodeName); err != nil {
		logrus.Errorf("%v", err)
		return err
	}

	if err := rotateCertificate(m.nodeName); err != nil {
		logrus.Errorf("%v", err)
		return err
	}

	if err := host.RestartKubelet(m.nodeName); err != nil {
		logrus.Errorf("%v", err)
		return err
	}

	return nil
}

// backupKubeconfig backups the kubconfigs in folder /etc/kubernetes
// which are issued by kubeadm
func backupKubeconfig(nodeName string) error {
	logrus.Infof("Commanding backup %s node kubeconfig", nodeName)

	kubeconfigs := []string{
		"/etc/kubernetes/admin.conf",
		"/etc/kubernetes/controller-manager.conf",
		"/etc/kubernetes/scheduler.conf",
	}

	var errors error
	for _, kubeconfig := range kubeconfigs {
		dir := filepath.Dir(kubeconfig)
		base := filepath.Base(kubeconfig)
		ext := filepath.Ext(base)
		backupKubeconfig := filepath.Join(dir, strings.TrimSuffix(base, ext)+"-"+time.Now().Format("20060102150405")+ext+".bak")

		// Relies on hostPID:true and privileged:true to enter host mount space
		var err error
		cmd := host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/cp", kubeconfig, backupKubeconfig)
		err = cmd.Run()
		if err != nil {
			errors = fmt.Errorf("%w; ", err)
			logrus.Fatalf("Error invoking %s: %v", cmd.Args, err)
		}
	}

	return errors
}

// backupCertificate backups the certificates folder /etc/kubernetes/pki
// issued by kubeadm
func backupCertificate(nodeName string) error {
	logrus.Infof("Commanding backup %s node certificate", nodeName)

	dir := "/etc/kubernetes/pki"
	backupDir := dir + "-" + time.Now().Format("20060102150405") + ".bak"

	// Relies on hostPID:true and privileged:true to enter host mount space
	var err error
	cmd := host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/cp", "--recursive", dir, backupDir)
	err = cmd.Run()
	if err != nil {
		logrus.Fatalf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}

// rotateCertificate calls kubeadm alpha certs renew all
// on the host system to rotates kubeadm issued certificates
func rotateCertificate(nodeName string) error {
	logrus.Infof("Commanding rotate %s node certificate", nodeName)

	// Relies on hostPID:true and privileged:true to enter host mount space
	var err error
	cmd := host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/kubeadm", "alpha", "certs", "renew", "all")
	err = cmd.Run()
	if err != nil {
		logrus.Fatalf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}
