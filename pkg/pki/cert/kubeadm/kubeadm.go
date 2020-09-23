/*
Copyright (c) 2020 SUSE LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubeadm

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jenting/kucero/pkg/host"
	"github.com/jenting/kucero/pkg/pki/cert"
	"github.com/jenting/kucero/pkg/pki/clock"
)

var certificates map[string]string = map[string]string{
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

type Kubeadm struct {
	nodeName           string
	expiryTimeToRotate time.Duration
	clock              clock.Clock
}

// New returns the kubeadm instance
func New(nodeName string, expiryTimeToRotate time.Duration) cert.Certificate {
	return &Kubeadm{
		nodeName:           nodeName,
		expiryTimeToRotate: expiryTimeToRotate,
		clock:              clock.NewRealClock(),
	}
}

// CheckExpiration checks control plane node certificate
// returns the certificates which are going to expires
func (k *Kubeadm) CheckExpiration() ([]string, error) {
	logrus.Infof("Commanding check %s node certificate expiration", k.nodeName)

	return kubeadmAlphaCertsCheckExpiration(k.expiryTimeToRotate, k.clock)
}

// Rotate executes the steps to rotates the certificate
// including backing up certificate, rotates certificate, and restart kubelet
func (k *Kubeadm) Rotate(expiryCertificates []string) error {
	var errs error
	for _, certificateName := range expiryCertificates {
		certificatePath, ok := certificates[certificateName]
		if !ok {
			continue
		}

		if err := backupCertificate(k.nodeName, certificateName, certificatePath); err != nil {
			errs = fmt.Errorf("%w; ", err)
			continue
		}

		if err := rotateCertificate(k.nodeName, certificateName, certificatePath); err != nil {
			errs = fmt.Errorf("%w; ", err)
			continue
		}
	}
	if errs != nil {
		return errs
	}

	if err := host.RestartKubelet(k.nodeName); err != nil {
		errs = fmt.Errorf("%w; ", err)
	}

	return errs
}

// backupCertificate backups the certificate/kubeconfig
// under folder /etc/kubernetes issued by kubeadm
func backupCertificate(nodeName string, certificateName, certificatePath string) error {
	logrus.Infof("Commanding backup %s node certificate %s path %s", nodeName, certificateName, certificatePath)

	dir := filepath.Dir(certificatePath)
	base := filepath.Base(certificatePath)
	ext := filepath.Ext(certificatePath)
	certificateBackupPath := filepath.Join(dir, strings.TrimSuffix(base, ext)+"-"+time.Now().Format("20060102030405")+ext+".bak")

	// Relies on hostPID:true and privileged:true to enter host mount space
	var err error
	cmd := host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/cp", certificatePath, certificateBackupPath)
	err = cmd.Run()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}

// rotateCertificate calls `kubeadm alpha certs renew <cert-name>`
// on the host system to rotates kubeadm issued certificates
func rotateCertificate(nodeName string, certificateName, certificatePath string) error {
	logrus.Infof("Commanding rotate %s node certificate %s path %s", nodeName, certificateName, certificatePath)

	err := kubeadmAlphaCertsRenew(certificateName, certificatePath)
	if err != nil {
		logrus.Errorf("Error invoking command: %v", err)
	}

	return err
}
