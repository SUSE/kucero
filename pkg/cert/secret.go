package cert

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"time"
)

type certMetric struct {
	notAfter        time.Time
	subject, issuer pkix.Name
}

func NewSecret() {

}

func secondsToExpiryFromCertAsFile(file string) (*certMetric, error) {
	certBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return secondsToExpiryFromCertAsBytes(certBytes)
}

func secondsToExpiryFromCertAsBase64String(s string) (*certMetric, error) {
	certBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	return secondsToExpiryFromCertAsBytes(certBytes)
}

func secondsToExpiryFromCertAsBytes(certBytes []byte) (*certMetric, error) {
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return nil, errors.New("Failed to parse as a pem")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &certMetric{
		notAfter: cert.NotAfter,
		subject:  cert.Subject,
		issuer:   cert.Issuer,
	}, nil
}
