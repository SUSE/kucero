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
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jenting/kucero/pkg/host"
)

var kubeadmCertificates map[string]string = map[string]string{
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

// kubeadmCheckExpiration executes `kubeadm alpha certs check-expiration`
// returns the certificates which are going to expires
func kubeadmCheckExpiration(expiryTimeToRotate time.Duration, clock Clock) ([]string, error) {
	expiryCertificates := []string{}

	// Relies on hostPID:true and privileged:true to enter host mount space
	cmd := host.NewCommandWithStdout("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/kubeadm", "alpha", "certs", "check-expiration")
	stdout, err := cmd.Output()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
		return expiryCertificates, err
	}

	stdoutS := string(stdout)
	kv := parseKubeadmCertsCheckExpiration(stdoutS)
	for cert, t := range kv {
		expiry := checkCertificateExpiry(cert, t, expiryTimeToRotate, clock)
		if expiry {
			expiryCertificates = append(expiryCertificates, cert)
		}
	}

	return expiryCertificates, nil
}

// parseKubeadmCertsCheckExpiration processes the `kubeadm alpha certs check-expiration`
// output and returns the certificate and expires information
func parseKubeadmCertsCheckExpiration(input string) map[string]time.Time {
	certExpires := make(map[string]time.Time)

	r := regexp.MustCompile("(.*) ([a-zA-Z]+ [0-9]{1,2}, [0-9]{4} [0-9]{1,2}:[0-9]{2} [a-zA-Z]+) (.*)")
	lines := strings.Split(input, "\n")
	parse := false
	for _, line := range lines {
		if parse {
			ss := r.FindStringSubmatch(line)
			if len(ss) < 3 {
				continue
			}

			cert := strings.TrimSpace(ss[1])
			t, err := time.Parse("Jan 02, 2006 15:04 MST", ss[2])
			if err != nil {
				fmt.Printf("err: %v\n", err)
				continue
			}

			certExpires[cert] = t
		}

		if strings.Contains(line, "CERTIFICATE") && strings.Contains(line, "EXPIRES") {
			parse = true
		}
	}

	return certExpires
}

func kubeadmRenewCerts(certificateName, certificatePath string) error {
	// Relies on hostPID:true and privileged:true to enter host mount space
	cmd := host.NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/kubeadm", "alpha", "certs", "renew", certificateName)
	return cmd.Run()
}
