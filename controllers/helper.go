/*
Copyright 2020 The cert-manager authors.

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

package controllers

import (
	"crypto/x509"
	"reflect"
	"strings"

	capi "k8s.io/api/certificates/v1"

	"github.com/sirupsen/logrus"
)

func hasExactUsages(csr *capi.CertificateSigningRequest, usages []capi.KeyUsage) bool {
	if len(usages) != len(csr.Spec.Usages) {
		return false
	}

	usageMap := map[capi.KeyUsage]struct{}{}
	for _, u := range usages {
		usageMap[u] = struct{}{}
	}

	for _, u := range csr.Spec.Usages {
		if _, ok := usageMap[u]; !ok {
			return false
		}
	}

	return true
}

var kubeletServerUsages = []capi.KeyUsage{
	capi.UsageKeyEncipherment,
	capi.UsageDigitalSignature,
	capi.UsageServerAuth,
}

func isNodeServingCert(csr *capi.CertificateSigningRequest, x509cr *x509.CertificateRequest) bool {
	if !reflect.DeepEqual([]string{"system:nodes"}, x509cr.Subject.Organization) {
		logrus.Warningf("Org does not match: %s", x509cr.Subject.Organization)
		return false
	}
	if (len(x509cr.DNSNames) < 1) || (len(x509cr.IPAddresses) < 1) {
		return false
	}
	if !hasExactUsages(csr, kubeletServerUsages) {
		logrus.Info("Usage does not match")
		return false
	}
	if !strings.HasPrefix(x509cr.Subject.CommonName, "system:node:") {
		logrus.Warningf("CN does not start with 'system:node': %s", x509cr.Subject.CommonName)
		return false
	}
	if csr.Spec.Username != x509cr.Subject.CommonName {
		logrus.Warningf("X509 CN %q doesn't match CSR username %q", x509cr.Subject.CommonName, csr.Spec.Username)
		return false
	}
	return true
}
