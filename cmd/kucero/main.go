package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/weaveworks/kured/pkg/daemonsetlock"
)

var (
	version = "unreleased"

	// Command line flags
	// kubeconfigPath                         string
	// includeCerts, excludeCerts             string
	// includeKubeconfigs, excludeKubeconfigs string
	pollingPeriod, expiryTimeToRotate   time.Duration
	dsNamespace, dsName, lockAnnotation string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kucero",
		Short: "KUbeadm CErtificate ROtation",
		Run:   root,
	}

	// rootCmd.PersistentFlags().StringVar(&includeCerts, "include-certs", "",
	// 	"File globs to include when looking for certs")
	// rootCmd.PersistentFlags().StringVar(&excludeCerts, "exclude-certs", "",
	// 	"File globs to exclude when looking for certs")
	// rootCmd.PersistentFlags().StringVar(&includeKubeconfigs, "include-kubeconfigs", "",
	// 	"File globs to include when looking for kubeconfigs")
	// rootCmd.PersistentFlags().StringVar(&excludeKubeconfigs, "exclude-kubeconfigs", "",
	// 	"File globs to exclude when looking for kubeconfigs")

	rootCmd.PersistentFlags().DurationVar(&pollingPeriod, "polling-period", time.Hour,
		"certificate rotation check period")
	rootCmd.PersistentFlags().DurationVar(&expiryTimeToRotate, "expiry-time-to-rotate", time.Hour*24*365,
		"rotates certificate when certificate less than expiry time")

	rootCmd.PersistentFlags().StringVar(&dsNamespace, "ds-namespace", "kube-system",
		"namespace containing daemonset on which to place lock")
	rootCmd.PersistentFlags().StringVar(&dsName, "ds-name", "kucero",
		"name of daemonset on which to place lock")
	rootCmd.PersistentFlags().StringVar(&lockAnnotation, "lock-annotation", "caasp.suse.com/kucero-node-lock",
		"annotation in which to record locking node")

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func root(cmd *cobra.Command, args []string) {
	logrus.Infof("KUbeadm CErtificate ROtation Daemon: %s", version)

	nodeName := os.Getenv("KUCERO_NODE_NAME")
	if nodeName == "" {
		logrus.Fatal("KUCERO_NODE_NAME environment variable required")
	}

	logrus.Infof("Node Name: %s", nodeName)
	logrus.Infof("Lock Annotation: %s/%s:%s", dsNamespace, dsName, lockAnnotation)
	logrus.Infof("Certificate Check Every %v", pollingPeriod)
	logrus.Infof("Rotates Certificate If Expiry Time Less Than %v", expiryTimeToRotate)

	rotateCertificateWhenNeeded(nodeName)
}

// nodeMeta is used to remember information across reboots
type nodeMeta struct {
	Unschedulable bool `json:"unschedulable"`
}

func rotateCertificateWhenNeeded(nodeName string) {
	config, err := rest.InClusterConfig()
	if err != nil {
		logrus.Fatal(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Fatal(err)
	}

	lock := daemonsetlock.New(client, nodeName, dsNamespace, dsName, lockAnnotation)

	nodeMeta := nodeMeta{}
	if holding(lock, &nodeMeta) {
		release(lock)
	}

	periodChannel := time.Tick(pollingPeriod)
	for {
		// TODO: rotate worker node's kubelet server certificate
		node, err := client.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
		if err != nil {
			logrus.Fatal(err)
		}

		labels := node.GetLabels()
		_, exist := labels["node-role.kubernetes.io/master="]
		if !exist {
			// logrus.Infof("Skipping worker nodes for now")
			continue
		}

		// check certificate expiration
		err = checkCertificateExpiration()
		if err != nil {
			logrus.Fatal(err)
		}

		// // load certificate
		// m, err := secondsToExpiryFromCertAsFile(includeCerts)
		// if err != nil {
		// 	logrus.Fatal(err)
		// }

		// // list the certs less than expiryTimeToRotate
		// // expiryTimeToRotate.Seconds()
		// // m.notAfter.Second()
		// elapsed := time.Until(m.notAfter)
		// fmt.Printf("%v\n", elapsed)
		// if elapsed > expiryTimeToRotate {
		// 	// elapsed time is greater than user setting expiryTimeToRotate
		// 	continue
		// }

		// // elapsed time is less than user setting expiryTimeToRotate
		// logrus.Infof("Going to roate certificate %s", m.subject.CommonName)

		// // find CA in the same directory path
		// dir, err := filepath.Abs(filepath.Dir(includeCerts))
		// if err != nil {
		// 	logrus.Fatal(err)
		// }
		// logrus.Infof("dir: %+v\n", dir)

		// caSignerFound := false
		// err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// 	if err != nil {
		// 		logrus.Warnf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
		// 		return err
		// 	}
		// 	if !info.IsDir() {
		// 		// load certificate
		// 		tmpCert, err := secondsToExpiryFromCertAsFile(path)
		// 		if err == nil {
		// 			// logrus.Infof("ok, try this one: %v\n", path)
		// 			if reflect.DeepEqual(m.issuer, tmpCert.subject) {
		// 				caSignerFound = true
		// 				logrus.Infof("Found CA signer cert path: %q\n", path)
		// 				return nil
		// 			}
		// 		}
		// 	}
		// 	return nil
		// })
		// if err != nil {
		// 	logrus.Warnf("error walking the path %q: %v\n", dir, err)
		// }
		// if !caSignerFound {
		// 	logrus.Warn("sorry cannot found ca")
		// }

		// backup certificate
		if err := backupKubeconfig(); err != nil {
			logrus.Errorf("%v\n", err)
		}
		if err := backupCertificate(); err != nil {
			logrus.Errorf("%v\n", err)
		}

		nodeMeta.Unschedulable = node.Spec.Unschedulable
		if acquire(lock, &nodeMeta) {
			rotateCertificate()
			for {
				logrus.Infof("Waiting for certificate rotation")
				time.Sleep(time.Minute)
			}
		}

		// rortate certificate
		if err := rotateCertificate(); err != nil {
			logrus.Errorf("%v\n", err)
		}

		// restart kubelet
		if err := restartKubelet(); err != nil {
			logrus.Errorf("%v\n", err)
		}

		<-periodChannel
	}
}

func checkCertificateExpiration() error {
	logrus.Info("Commanding check certificate expiration")

	// Relies on hostPID:true and privileged:true to enter host mount space
	cmd := newCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/kubeadm", "alpha", "certs", "check-expiration")
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", cmd.Stderr)
	}

	fmt.Printf("%s\n", cmd.Stdout)

	return nil
}

func backupKubeconfig() error {
	logrus.Info("Commanding backup kubeconfig")

	kubeconfigs := []string{
		"/etc/kubernetes/admin.conf",
		"/etc/kubernetes/controller-manager.conf",
		"/etc/kubernetes/scheduler.conf",
	}

	var errors error
	for _, kubeconfig := range kubeconfigs {
		basename := filepath.Base(kubeconfig)
		ext := filepath.Ext(basename)
		backupKubeconfigPath := strings.TrimSuffix(basename, ext) + "-" + time.Now().Format("20060102150405") + ext + ".bak"

		// Relies on hostPID:true and privileged:true to enter host mount space
		cmd := newCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/cp", "-rf", kubeconfig, backupKubeconfigPath)
		err := cmd.Run()
		if err != nil {
			errors = fmt.Errorf("%w; ", err)
			logrus.Fatalf("Error invoking cp -rf %s %s command: %v", kubeconfig, backupKubeconfigPath, err)
		}
	}

	return errors
}

func backupCertificate() error {
	logrus.Info("Commanding backup certificate")

	dirPath := "/etc/kubernetes/pki"
	backupDirPath := dirPath + "-" + time.Now().Format("20060102150405") + ".bak"

	// Relies on hostPID:true and privileged:true to enter host mount space
	return newCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/cp", "-rf", dirPath, backupDirPath).Run()
}

func rotateCertificate() error {
	logrus.Info("Commanding certificate rotate")

	// Relies on hostPID:true and privileged:true to enter host mount space
	return newCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/kubeadm", "alpha", "certs", "renew", "all").Run()
}

func restartKubelet() error {
	logrus.Info("Commanding restart kubelet")

	// Relies on hostPID:true and privileged:true to enter host mount space
	return newCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/systemctl", "restart", "kubelet").Run()
}

// newCommand creates a new Command with stdout/stderr wired to our standard logger
func newCommand(name string, arg ...string) *exec.Cmd {
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

func holding(lock *daemonsetlock.DaemonSetLock, metadata interface{}) bool {
	holding, err := lock.Test(metadata)
	if err != nil {
		logrus.Fatalf("Error testing lock: %v", err)
	}
	if holding {
		logrus.Infof("Holding lock")
	}
	return holding
}

func acquire(lock *daemonsetlock.DaemonSetLock, metadata interface{}) bool {
	holding, holder, err := lock.Acquire(metadata)
	switch {
	case err != nil:
		logrus.Fatalf("Error acquiring lock: %v", err)
		return false
	case !holding:
		logrus.Warnf("Lock already held: %v", holder)
		return false
	default:
		logrus.Infof("Acquired reboot lock")
		return true
	}
}

func release(lock *daemonsetlock.DaemonSetLock) {
	logrus.Infof("Releasing lock")
	if err := lock.Release(); err != nil {
		logrus.Fatalf("Error releasing lock: %v", err)
	}
}
