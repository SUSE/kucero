package host

import (
	"github.com/sirupsen/logrus"
)

func Uncordon(nodeName string) error {
	logrus.Infof("Uncordoning %s node", nodeName)

	cmd := NewCommand("/usr/bin/kubectl", "uncordon", nodeName)
	err := cmd.Run()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}

func Cordon(nodeName string) error {
	logrus.Infof("Cordoning %s node", nodeName)

	cmd := NewCommand("/usr/bin/kubectl", "cordon", nodeName)
	err := cmd.Run()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}

func Drain(nodeName string) error {
	logrus.Infof("Draining %s node", nodeName)

	cmd := NewCommand("/usr/bin/kubectl", "drain", "--ignore-daemonsets", "--delete-local-data", "--force", nodeName)
	err := cmd.Run()
	if err != nil {
		logrus.Errorf("Error invoking %s: %v", cmd.Args, err)
	}

	return err
}
