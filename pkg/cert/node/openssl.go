package node

import (
	"errors"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// parseOpensslEnddate processes the `openssl x509 -noout -enddate -in <certificate-path>`
// output and returns the expires information
func parseOpensslEnddate(input string) (*time.Time, error) {
	// notAfter=Jan 02 15:04:05 2006 MST
	if !strings.Contains(input, "notAfter") {
		err := errors.New("Cannot found notAfter key")
		logrus.Errorf("%v", err)
		return nil, err
	}

	ss := strings.Split(input, "=")
	if len(ss) < 2 {
		err := errors.New("Cannot found enddate")
		logrus.Errorf("%v", err)
		return nil, err
	}

	t, err := time.Parse("Jan 2 15:04:05 2006 MST", ss[1])
	if err != nil {
		logrus.Errorf("%v", err)
		return nil, err
	}

	return &t, nil
}

func parseOpensslSubjectDN(input string) ([]string, error) {
	// subject=CN = master0
	if !strings.Contains(input, "subject") {
		err := errors.New("Cannot found subject key")
		logrus.Errorf("%v", err)
		return nil, err
	}

	ss := strings.SplitN(input, "=", 2)
	if len(ss) < 2 {
		err := errors.New("Cannot found subject")
		logrus.Errorf("%v", err)
		return nil, err
	}
	subject := strings.TrimRight(ss[1], "\n")
	return strings.Split(subject, ", "), nil
}

func parseOpensslSubjectAltName(input string) ([]string, []string) {
	var dns, ipAddress []string

	lines := strings.Split(input, "\n")
	for _, line := range lines {
		if strings.Contains(line, "DNS:") || strings.Contains(line, "IP Address:") {
			s := strings.Split(line, ", ")
			for _, ss := range s {
				if strings.Contains(ss, "DNS:") {
					sss := strings.Split(ss, ":")
					dns = append(dns, sss[1])
				}
				if strings.Contains(ss, "IP Address:") {
					sss := strings.SplitN(ss, ":", 2)
					ipAddress = append(ipAddress, sss[1])
				}
			}
		}
	}

	return dns, ipAddress
}
