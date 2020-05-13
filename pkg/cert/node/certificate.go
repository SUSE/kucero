package node

import (
	"time"

	"github.com/sirupsen/logrus"
)

type OWNER string

const (
	kubeadm OWNER = "kubeadm"
	kubelet OWNER = "kubelet"
)

type Certificate interface {
	// CheckExpiration checks node certificate
	// returns the certificates which are going to expires
	CheckExpiration() (map[OWNER][]string, error)

	// Rotate rotates the node certificates
	// which are going to expires
	Rotate(expiryCertificates map[OWNER][]string) error
}

// checkExpiry checks if the time `t` is less than the time duration `expiryTimeToRotate`
func checkExpiry(name string, t time.Time, expiryTimeToRotate time.Duration, clock Clock) bool {
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
