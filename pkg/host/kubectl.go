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
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	kubectldrain "k8s.io/kubectl/pkg/drain"

	"github.com/sirupsen/logrus"
)

// Uncordon executes `kubectl uncordon <node-name>`
// on the host system
func Uncordon(client *kubernetes.Clientset, corev1Node *corev1.Node) error {
	nodeName := corev1Node.GetName()
	logrus.Infof("Uncordoning %s node", nodeName)

	drainer := &kubectldrain.Helper{
		Client: client,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	// RunCordonOrUncordon runs either Cordon or Uncordon.
	// The desired value "false" is passed to "Unschedulable" to indicate that the node is schedulable.
	if err := kubectldrain.RunCordonOrUncordon(drainer, corev1Node, false); err != nil {
		logrus.Errorf("Error uncordonning %s: %v", nodeName, err)
	}
	return nil
}

// Cordon executes `kubectl cordon <node-name>`
// on the host system
func Cordon(client *kubernetes.Clientset, corev1Node *corev1.Node) error {
	nodeName := corev1Node.GetName()
	logrus.Infof("Cordoning %s node", nodeName)

	drainer := &kubectldrain.Helper{
		Client: client,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	// RunCordonOrUncordon runs either Cordon or Uncordon.
	// The desired value "true" is passed to "Unschedulable" to indicate that the node is unschedulable.
	if err := kubectldrain.RunCordonOrUncordon(drainer, corev1Node, true); err != nil {
		logrus.Errorf("Error cordonning %s: %v", nodeName, err)
	}
	return nil
}

// Drain executes `kubectl drain --ignore-daemonsets --delete-local-data --force <node-name>`
// on the host system
func Drain(client *kubernetes.Clientset, corev1Node *corev1.Node) error {
	nodeName := corev1Node.GetName()
	logrus.Infof("Draining %s node", nodeName)

	drainer := &kubectldrain.Helper{
		Client:              client,
		Force:               true,
		DeleteLocalData:     true,
		IgnoreAllDaemonSets: true,
		Out:                 os.Stdout,
		ErrOut:              os.Stderr,
	}
	if err := kubectldrain.RunNodeDrain(drainer, nodeName); err != nil {
		logrus.Errorf("Error draining %s: %v", nodeName, err)
	}
	return nil
}
