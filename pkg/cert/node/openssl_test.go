package node

import (
	"reflect"
	"testing"
	"time"
)

func Test_parseKubeletCheckExpiration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		expect    time.Time
	}{
		{
			name:   "normal case",
			input:  "notAfter=Jan 02 15:04:05 2006 UTC",
			expect: time.Date(2006, time.January, 02, 15, 04, 05, 00, time.UTC),
		},
		{
			name:      "without notAfter key",
			input:     "Jan 02 15:04:05 2006 UTC",
			expectErr: true,
		},
		{
			name:      "without equal sign",
			input:     "notAfter",
			expectErr: true,
		},
		{
			name:      "time format incorrect",
			input:     "notAfter=Jan 02 15:04:05 2006",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotT, err := parseOpensslEnddate(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("Expect error but got no error")
				}
				return
			}

			if !reflect.DeepEqual(*gotT, tt.expect) {
				t.Errorf("gotT %v is not equals to expected %v", *gotT, tt.expect)
			}
		})
	}
}

func Test_parseOpensslSubjectDN(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		expect    []string
	}{
		{
			name:   "short case",
			input:  "subject=CN = master0",
			expect: []string{"CN = master0"},
		},
		{
			name:   "long case",
			input:  "subject=C = TW, ST = Taiwan, L = Hsinchu, O = Zhubei, OU = 101, CN = zhubei101",
			expect: []string{"C = TW", "ST = Taiwan", "L = Hsinchu", "O = Zhubei", "OU = 101", "CN = zhubei101"},
		},
		{
			name:      "without subject key",
			input:     "Jan 02 15:04:05 2006 UTC",
			expectErr: true,
		},
		{
			name:      "without equal sign",
			input:     "subject",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOpensslSubjectDN(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("Expect error but got no error")
				}
				return
			}

			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("got %s is not equals to expected %s", got, tt.expect)
			}
		})
	}
}

func Test_parseOpensslSubjectAltName(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectDNS       []string
		expectIPAddress []string
	}{
		{
			name: "dns case",
			input: `
X509v3 extensions:
	X509v3 Key Usage: critical
		Digital Signature, Key Encipherment
	X509v3 Extended Key Usage:
		TLS Web Server Authentication
	X509v3 Subject Alternative Name:
		DNS:master0, DNS:kubernetes, DNS:kubernetes.default, DNS:kubernetes.default.svc, DNS:kubernetes.default.svc.cluster.local
			`,
			expectDNS: []string{"master0", "kubernetes", "kubernetes.default", "kubernetes.default.svc", "kubernetes.default.svc.cluster.local"},
		},
		{
			name: "ip case",
			input: `
X509v3 extensions:
	X509v3 Key Usage: critical
		Digital Signature, Key Encipherment
	X509v3 Extended Key Usage:
		TLS Web Server Authentication
	X509v3 Subject Alternative Name:
		IP Address:10.96.0.1, IP Address:10.84.72.60, IP Address:10.84.72.60, IP Address:10.84.72.60
			`,
			expectIPAddress: []string{"10.96.0.1", "10.84.72.60", "10.84.72.60", "10.84.72.60"},
		},
		{
			name: "both dns and ip case",
			input: `
X509v3 extensions:
	X509v3 Key Usage: critical
		Digital Signature, Key Encipherment
	X509v3 Extended Key Usage:
		TLS Web Server Authentication
	X509v3 Subject Alternative Name:
		DNS:master0, DNS:kubernetes, DNS:kubernetes.default, DNS:kubernetes.default.svc, DNS:kubernetes.default.svc.cluster.local, IP Address:10.96.0.1, IP Address:10.84.72.60, IP Address:10.84.72.60, IP Address:10.84.72.60
			`,
			expectDNS:       []string{"master0", "kubernetes", "kubernetes.default", "kubernetes.default.svc", "kubernetes.default.svc.cluster.local"},
			expectIPAddress: []string{"10.96.0.1", "10.84.72.60", "10.84.72.60", "10.84.72.60"},
		},
		{
			name: "ipv4 and ipv6 case",
			input: `
X509v3 extensions:
	X509v3 Key Usage: critical
		Digital Signature, Key Encipherment
	X509v3 Extended Key Usage:
		TLS Web Server Authentication
	X509v3 Subject Alternative Name:
		IP Address:10.84.73.15, IP Address:127.0.0.1, IP Address:0:0:0:0:0:0:0:1, IP Address:10.84.73.15
			`,
			expectIPAddress: []string{"10.84.73.15", "127.0.0.1", "0:0:0:0:0:0:0:1", "10.84.73.15"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotDNS, gotIPAddress := parseOpensslSubjectAltName(tt.input)

			if !reflect.DeepEqual(gotDNS, tt.expectDNS) {
				t.Errorf("got DNS %s is not equals to expected DNS %s", gotDNS, tt.expectDNS)
			}
			if !reflect.DeepEqual(gotIPAddress, tt.expectIPAddress) {
				t.Errorf("got IP Address %s is not equals to expected IP Address %s", gotIPAddress, tt.expectIPAddress)
			}
		})
	}
}
