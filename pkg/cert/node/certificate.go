package node

import (
	"time"

	"github.com/sirupsen/logrus"
)

type Certificate interface {
	// CheckExpiration checks node certificate
	// and return the certificates which are going to expiry
	CheckExpiration() ([]string, error)

	// Rotate rotates the node certificates
	Rotate(expiryCertificates []string) error
}

func CheckExpiry(name string, t time.Time, expiryTimeToRotate time.Duration) bool {
	tn := time.Now()
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
