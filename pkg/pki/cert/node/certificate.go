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

package node

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jenting/kucero/pkg/host"
)

type Certificate interface {
	// CheckExpiration checks node certificate
	// returns the certificates which are going to expires
	CheckExpiration() ([]string, error)

	// Rotate rotates the node certificates
	// which are going to expires
	Rotate(expiryCertificates []string) error
}

// checkCertificateExpiry checks if the time `t` is less than the time duration `expiryTimeToRotate`
func checkCertificateExpiry(name string, t time.Time, expiryTimeToRotate time.Duration, clock Clock) bool {
	tn := clock.Now()
	if t.Before(tn) {
		logrus.Infof("The certificate %s is expiry already", name)
		return true
	} else if t.Sub(tn) <= expiryTimeToRotate {
		logrus.Infof("The certificate %s notAfter is less than user specified expiry time %s", name, expiryTimeToRotate)
		return true
	}

	logrus.Infof("The certificate %s is still valid for %s", name, t.Sub(tn))
	return false
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

	err := kubeadmRenewCerts(certificateName, certificatePath)
	if err != nil {
		logrus.Errorf("Error invoking command: %v", err)
	}

	return err
}
