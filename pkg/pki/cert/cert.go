package cert

import (
	"crypto/x509"
	"encoding/pem"
	"errors"

	capi "k8s.io/api/certificates/v1beta1"
)

// IsCertificateRequestApproved returns true if a certificate request has the
// "Approved" condition and no "Denied" conditions; false otherwise.
func IsCertificateRequestApproved(csr *capi.CertificateSigningRequest) bool {
	approved, denied := GetCertApprovalCondition(&csr.Status)
	return approved && !denied
}

func GetCertApprovalCondition(status *capi.CertificateSigningRequestStatus) (approved bool, denied bool) {
	for _, c := range status.Conditions {
		if c.Type == capi.CertificateApproved {
			approved = true
		}
		if c.Type == capi.CertificateDenied {
			denied = true
		}
	}
	return
}

// ParseCSR decodes a PEM encoded CSR
// Copied from https://github.com/kubernetes/kubernetes/blob/v1.18.0-beta.2/pkg/apis/certificates/v1beta1/helpers.go
func ParseCSR(pemBytes []byte) (*x509.CertificateRequest, error) {
	// extract PEM from request object
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return nil, errors.New("PEM block type must be CERTIFICATE REQUEST")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, err
	}
	return csr, nil
}
