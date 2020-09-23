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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

type action struct {
	check  func(*Kubelet, string) (bool, error)
	update func(*Kubelet, string, string) error
}

var configs map[string]action = map[string]action{
	"/etc/kubernetes/kubelet.conf": {
		check: func(k *Kubelet, filepath string) (bool, error) {
			return k.checkEtcKubernetesKubeletConf(filepath)
		},
		update: func(k *Kubelet, oldFilepath, newFilepath string) error {
			return k.updateEtcKubernetesKubeletConf(oldFilepath, newFilepath)
		},
	},
	"/var/lib/kubelet/config.yaml": {
		check: func(k *Kubelet, filepath string) (bool, error) {
			return k.checkVarLibKubeletConfigYaml(filepath)
		},
		update: func(k *Kubelet, oldFilepath, newFilepath string) error {
			return k.updateVarLibKubeletConfigYaml(oldFilepath, newFilepath)
		},
	},
}

// checkEtcKubernetesKubeletConf checks /etc/kubernetes/kubelet.conf need to be update
// if client-certificate-data or client-key-data exist
func (k *Kubelet) checkEtcKubernetesKubeletConf(filepath string) (bool, error) {
	kubeletConfig, err := clientcmd.LoadFromFile(filepath)
	if err != nil {
		return false, err
	}

	for _, authInfo := range kubeletConfig.AuthInfos {
		if len(authInfo.ClientKeyData) > 0 || len(authInfo.ClientCertificateData) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// updateEtcKubernetesKubeletConf updates the /etc/kubernetes/kubelet.conf
// from
//   client-certificate-data: <base64-encoded-client-certificate>
//   client-key-data: <base64-encoded-client-key>
// to
//   client-certificate: /var/lib/kubelet/pki/kubelet-client-current.pem
//   client-key: /var/lib/kubelet/pki/kubelet-client-current.pem
func (k *Kubelet) updateEtcKubernetesKubeletConf(oldFilepath, newFilepath string) error {
	kubeletConfig, err := clientcmd.LoadFromFile(oldFilepath)
	if err != nil {
		return err
	}

	for _, authInfo := range kubeletConfig.AuthInfos {
		if len(authInfo.ClientKeyData) > 0 || len(authInfo.ClientCertificateData) > 0 {
			authInfo.ClientKeyData = []byte{}
			authInfo.ClientCertificateData = []byte{}
			authInfo.ClientKey = "/var/lib/kubelet/pki/kubelet-client-current.pem"
			authInfo.ClientCertificate = "/var/lib/kubelet/pki/kubelet-client-current.pem"
		}
	}

	if err = clientcmd.WriteToFile(*kubeletConfig, newFilepath); err != nil {
		return fmt.Errorf("failed to serialize %q", newFilepath)
	}
	return nil
}

// kubeletConfiguration contains the configuration for the /var/lib/kubelet/config.yaml
type kubeletConfiguration struct {
	RotateCertificates bool `json:"rotateCertificates,omitempty"`
	ServerTLSBootstrap bool `json:"serverTLSBootstrap,omitempty"`
}

// checkVarLibKubeletConfigYaml checks /var/lib/kubelet/config.yaml need to be update
// if rotateCertificates and serverTLSBootstrap does not match the configuration
func (k *Kubelet) checkVarLibKubeletConfigYaml(filepath string) (bool, error) {
	kubeletConfig, err := ioutil.ReadFile(filepath)
	if err != nil {
		return false, err
	}

	kc := kubeletConfiguration{}
	if err := yaml.Unmarshal(kubeletConfig, &kc); err != nil {
		return false, err
	}

	if kc.RotateCertificates != k.enableKubeletClientCertRotation {
		return true, nil
	}
	if kc.ServerTLSBootstrap != k.enableKubeletServerCertRotation {
		return true, nil
	}
	return false, nil
}

// updateVarLibKubeletConfigYaml updates /var/lib/kubelet/config.yaml of
// the key `rotateCertificates` and `serverTLSBootstrap`
func (k *Kubelet) updateVarLibKubeletConfigYaml(oldFilepath, newFilepath string) error {
	kubeletConfig, err := ioutil.ReadFile(oldFilepath)
	if err != nil {
		return err
	}

	kc := kubeletConfiguration{}
	if err := yaml.Unmarshal(kubeletConfig, &kc); err != nil {
		return err
	}

	if kc.RotateCertificates != k.enableKubeletClientCertRotation {
		// set "rotateCertificates: true" or "rotateCertificates: false" in /var/lib/kubelet/config.yaml to enable/disable kubelet client cert rotation
		if bytes.Contains(kubeletConfig, []byte("rotateCertificates:")) {
			kubeletConfig = bytes.Replace(kubeletConfig,
				[]byte(fmt.Sprintf("rotateCertificates: %t", kc.RotateCertificates)),
				[]byte(fmt.Sprintf("rotateCertificates: %t", k.enableKubeletClientCertRotation)),
				1)
		} else {
			kubeletConfig = append(kubeletConfig, []byte(fmt.Sprintf("rotateCertificates: %t\n", k.enableKubeletClientCertRotation))...)
		}
	}
	if kc.ServerTLSBootstrap != k.enableKubeletServerCertRotation {
		// set "serverTLSBootstrap: true" or "serverTLSBootstrap: false" in /var/lib/kubelet/config.yaml to enable/disable kubelet server cert rotation
		if bytes.Contains(kubeletConfig, []byte("serverTLSBootstrap:")) {
			kubeletConfig = bytes.Replace(kubeletConfig,
				[]byte(fmt.Sprintf("serverTLSBootstrap: %t", kc.ServerTLSBootstrap)),
				[]byte(fmt.Sprintf("serverTLSBootstrap: %t", k.enableKubeletServerCertRotation)),
				1)
		} else {
			kubeletConfig = append(kubeletConfig, []byte(fmt.Sprintf("serverTLSBootstrap: %t\n", k.enableKubeletServerCertRotation))...)
		}
	}

	f, err := os.Stat(oldFilepath)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(newFilepath, kubeletConfig, f.Mode())
}
