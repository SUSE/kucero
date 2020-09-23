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
	certificates := make(map[string]string)
	for k, v := range kubeadmCertificates {
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
func (m *Master) CheckExpiration() ([]string, error) {
	logrus.Infof("Commanding check %s node certificate expiration", m.nodeName)

	return kubeadmCheckExpiration(m.expiryTimeToRotate, m.clock)
}

// Rotate executes the steps to rotates the certificate
// including backing up certificate, rotates certificate, and restart kubelet
func (m *Master) Rotate(expiryCertificates []string) error {
	var errs error
	for _, certificateName := range expiryCertificates {
		certificatePath, ok := m.certificates[certificateName]
		if !ok {
			continue
		}

		if err := backupCertificate(m.nodeName, certificateName, certificatePath); err != nil {
			errs = fmt.Errorf("%w; ", err)
			continue
		}

		if err := rotateCertificate(m.nodeName, certificateName, certificatePath); err != nil {
			errs = fmt.Errorf("%w; ", err)
			continue
		}
	}
	if errs != nil {
		return errs
	}

	if err := host.RestartKubelet(m.nodeName); err != nil {
		errs = fmt.Errorf("%w; ", err)
	}

	return errs
}
