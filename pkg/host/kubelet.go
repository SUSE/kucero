package host

import "github.com/sirupsen/logrus"

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
