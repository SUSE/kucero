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

package host

import "github.com/sirupsen/logrus"

// RestartKubelet executes `systemctl restart kubelet`
// on the host system
func RestartKubelet(nodeName string) error {
	logrus.Infof("Commanding restart kubelet on %s node", nodeName)

	// Relies on hostPID:true and privileged:true to enter host mount space
	cmd := NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/systemctl", "restart", "kubelet")
	err := cmd.Run()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}
