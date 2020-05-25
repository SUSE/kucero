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

import (
	"github.com/sirupsen/logrus"
)

// Uncordon executes `kubectl uncordon <node-name>`
// on the host system
func Uncordon(nodeName string) error {
	logrus.Infof("Uncordoning %s node", nodeName)

	cmd := NewCommand("/usr/bin/kubectl", "uncordon", nodeName)
	err := cmd.Run()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}

// Cordon executes `kubectl cordon <node-name>`
// on the host system
func Cordon(nodeName string) error {
	logrus.Infof("Cordoning %s node", nodeName)

	cmd := NewCommand("/usr/bin/kubectl", "cordon", nodeName)
	err := cmd.Run()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}

// Drain executes `kubectl drain --ignore-daemonsets --delete-local-data --force <node-name>`
// on the host system
func Drain(nodeName string) error {
	logrus.Infof("Draining %s node", nodeName)

	cmd := NewCommand("/usr/bin/kubectl", "drain", "--ignore-daemonsets", "--delete-local-data", "--force", nodeName)
	err := cmd.Run()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}
