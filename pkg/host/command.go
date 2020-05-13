package host

import (
	"os/exec"

	"github.com/sirupsen/logrus"
)

// NewCommand creates a new Command with stdout/stderr wired to our standard logger
func NewCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)

	cmd.Stdout = logrus.NewEntry(logrus.StandardLogger()).
		WithField("cmd", cmd.Args[0]).
		WithField("std", "out").
		WriterLevel(logrus.InfoLevel)

	cmd.Stderr = logrus.NewEntry(logrus.StandardLogger()).
		WithField("cmd", cmd.Args[0]).
		WithField("std", "err").
		WriterLevel(logrus.WarnLevel)

	return cmd
}

// NewCommand creates a new Command with stderr wired to our standard logger
func NewCommandWithStdout(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)

	cmd.Stderr = logrus.NewEntry(logrus.StandardLogger()).
		WithField("cmd", cmd.Args[0]).
		WithField("std", "err").
		WriterLevel(logrus.WarnLevel)

	return cmd
}
