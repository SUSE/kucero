/*
Copyright 2019 The Kubernetes Authors.

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

// Package signer implements a CA signer that uses keys stored on local disk.
package signer

import (
	"crypto/x509"
	"encoding/pem"
	"time"

	capi "k8s.io/api/certificates/v1beta1"

	"github.com/jenting/kucero/pkg/pki/authority"
)

type Signer struct {
	caProvider *caProvider
	certTTL    time.Duration
}

func NewSigner(caFile, caKeyFile string, duration time.Duration) (*Signer, error) {
	caProvider, err := newCAProvider(caFile, caKeyFile)
	if err != nil {
		return nil, err
	}

	ret := &Signer{
		caProvider: caProvider,
		certTTL:    duration,
	}
	return ret, nil
}

func (s *Signer) Sign(x509cr *x509.CertificateRequest, usages []capi.KeyUsage) ([]byte, error) {
	currCA, err := s.caProvider.currentCA()
	if err != nil {
		return nil, err
	}
	der, err := currCA.Sign(x509cr.Raw, authority.PermissiveSigningPolicy{
		TTL:    s.certTTL,
		Usages: usages,
	})
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), nil
}
