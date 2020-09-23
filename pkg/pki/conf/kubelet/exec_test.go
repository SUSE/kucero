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

package kubelet

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"
)

func TestEtcKubernetesKubeletConf(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		expect   bool
	}{
		{
			name:     "kubelet.conf before Kubernetes version 1.17",
			filepath: "tests/1.16-kubelet.conf",
			expect:   true,
		},
		{
			name:     "kubelet.conf after Kubernetes version 1.17",
			filepath: "tests/1.17-kubelet.conf",
			expect:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			k := &Kubelet{nodeName: tt.name}
			got, err := k.checkEtcKubernetesKubeletConf(tt.filepath)
			if err != nil {
				t.Errorf("expected no error but error reported: %v\n", err)
			}

			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("got %v is not equals to expected", got)
			}

			// create a temporary file to save the updated kubelet.conf
			file, err := ioutil.TempFile("tests", "*-kubelet.yaml")
			if err != nil {
				log.Fatal(err)
			}
			defer os.Remove(file.Name())

			err = k.updateEtcKubernetesKubeletConf(tt.filepath, file.Name())
			if err != nil {
				t.Errorf("expected no error but error reported: %v\n", err)
			}

			got, err = k.checkEtcKubernetesKubeletConf(file.Name())
			if err != nil {
				t.Errorf("expected no error but error reported: %v\n", err)
			}
			if got {
				t.Errorf("expected no update but a update reported: %t\n", got)
			}
		})
	}
}

func TestVarLibKubeletConfigYaml(t *testing.T) {
	tests := []struct {
		name                            string
		filepath                        string
		enableKubeletClientCertRotation bool
		enableKubeletServerCertRotation bool
		expect                          bool
	}{
		{
			name:                            "client: client enabled, server enabled",
			filepath:                        "tests/client-kubelet.yaml",
			enableKubeletClientCertRotation: true,
			enableKubeletServerCertRotation: true,
			expect:                          true,
		},
		{
			name:                            "client: client enabled, server disabled",
			filepath:                        "tests/client-kubelet.yaml",
			enableKubeletClientCertRotation: true,
			enableKubeletServerCertRotation: false,
			expect:                          false,
		},
		{
			name:                            "client: client disabled, server enabled",
			filepath:                        "tests/client-kubelet.yaml",
			enableKubeletClientCertRotation: false,
			enableKubeletServerCertRotation: true,
			expect:                          true,
		},
		{
			name:                            "client: client disabled, server disabled",
			filepath:                        "tests/client-kubelet.yaml",
			enableKubeletClientCertRotation: false,
			enableKubeletServerCertRotation: false,
			expect:                          true,
		},
		{
			name:                            "server: client enabled, server enabled",
			filepath:                        "tests/server-kubelet.yaml",
			enableKubeletClientCertRotation: true,
			enableKubeletServerCertRotation: true,
			expect:                          true,
		},
		{
			name:                            "server: client enabled, server disabled",
			filepath:                        "tests/server-kubelet.yaml",
			enableKubeletClientCertRotation: true,
			enableKubeletServerCertRotation: false,
			expect:                          true,
		},
		{
			name:                            "server: client disabled, server enabled",
			filepath:                        "tests/server-kubelet.yaml",
			enableKubeletClientCertRotation: false,
			enableKubeletServerCertRotation: true,
			expect:                          false,
		},
		{
			name:                            "server: client disabled, server disabled",
			filepath:                        "tests/server-kubelet.yaml",
			enableKubeletClientCertRotation: false,
			enableKubeletServerCertRotation: false,
			expect:                          true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			k := &Kubelet{
				nodeName:                        tt.name,
				enableKubeletClientCertRotation: tt.enableKubeletClientCertRotation,
				enableKubeletServerCertRotation: tt.enableKubeletServerCertRotation,
			}
			got, err := k.checkVarLibKubeletConfigYaml(tt.filepath)
			if err != nil {
				t.Errorf("expected no error but error reported: %v\n", err)
			}

			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("got %v is not equals to expected", got)
			}

			// create a temporary file to save the updated kubelet.yaml
			file, err := ioutil.TempFile("tests", "*-kubelet.yaml")
			if err != nil {
				log.Fatal(err)
			}
			defer os.Remove(file.Name())

			err = k.updateVarLibKubeletConfigYaml(tt.filepath, file.Name())
			if err != nil {
				t.Errorf("expected no error but error reported: %v\n", err)
			}

			got, err = k.checkVarLibKubeletConfigYaml(file.Name())
			if err != nil {
				t.Errorf("expected no error but error reported: %v\n", err)
			}
			if got {
				t.Errorf("expected no update but a update reported: %t\n", got)
			}
		})
	}
}
