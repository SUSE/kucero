package host

import "github.com/sirupsen/logrus"

func RestartKubelet(nodeName string) error {
	logrus.Infof("Commanding restart kubelet on %s node", nodeName)

	// Relies on hostPID:true and privileged:true to enter host mount space
	cmd := NewCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/systemctl", "restart", "kubelet")
	if err := cmd.Run(); err != nil {
		logrus.Fatalf("Error invoking command: %v", cmd.Args)
		return err
	}
	return nil
}
