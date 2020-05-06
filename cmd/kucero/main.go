package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/weaveworks/kured/pkg/daemonsetlock"
)

var (
	version = "unreleased"

	// Command line flags
	kubeconfigPath     string
	pollingPeriod      time.Duration
	dsNamespace        string
	dsName             string
	lockAnnotation     string
	certExporterURL    string
	expiryTimeToRotate time.Duration
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kucero",
		Short: "KUbernetes CErtificate ROtation",
		Run:   root,
	}

	rootCmd.PersistentFlags().StringVar(&kubeconfigPath, "kubeconfig", "",
		"path to the kubeconfig. Only required if out-of-cluster.")

	rootCmd.PersistentFlags().DurationVar(&pollingPeriod, "polling-period", time.Hour,
		"certificate rotation check period")
	rootCmd.PersistentFlags().StringVar(&certExporterURL, "cert-exporter-url", "localhost:8080/metrics",
		"cert-exporter instance to probe for certificate information")
	rootCmd.PersistentFlags().DurationVar(&expiryTimeToRotate, "expiry-time-to-rotate", time.Hour,
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
	logrus.Infof("KUbernetes CErtificate ROtation Daemon: %s", version)

	nodeID := os.Getenv("KUCERO_NODE_ID")
	if nodeID == "" {
		logrus.Fatal("KUCERO_NODE_ID environment variable required")
	}

	logrus.Infof("Node ID: %s", nodeID)
	logrus.Infof("Lock Annotation: %s/%s:%s", dsNamespace, dsName, lockAnnotation)
	logrus.Infof("Certificate Check Every %v On URL %s", pollingPeriod, certExporterURL)
	logrus.Infof("Rotates Certificate If Expiry Time Less Than %v", expiryTimeToRotate)

	rotateCertificateWhenNeeded(nodeID)
}

// nodeMeta is used to remember information across reboots
type nodeMeta struct {
	Unschedulable bool `json:"unschedulable"`
}

func rotateCertificateWhenNeeded(nodeID string) {
	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	logrus.Fatal(err)
	// }
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		logrus.Fatal(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Fatal(err)
	}

	lock := daemonsetlock.New(client, nodeID, dsNamespace, dsName, lockAnnotation)

	nodeMeta := nodeMeta{}
	if holding(lock, &nodeMeta) {
		release(lock)
	}

	periodChannel := time.Tick(pollingPeriod)
	for {
		// read prometheus URL cert_exporter metrics
		resp, err := http.Get(certExporterURL)
		if err != nil {
			logrus.Fatal(err)
		}
		fmt.Printf("%v\n", resp)

		// list the certs less than expiryTimeToRotate

		// rotate certificates

		node, err := client.CoreV1().Nodes().Get(nodeID, metav1.GetOptions{})
		if err != nil {
			logrus.Fatal(err)
		}
		_, exist := node.GetLabels()["node-role.kubernetes.io/master="]
		if !exist {
			logrus.Infof("Skipping worker nodes")
			continue
		}

		nodeMeta.Unschedulable = node.Spec.Unschedulable
		if acquire(lock, &nodeMeta) {
			rotateCertificate(nodeID)
			for {
				logrus.Infof("Waiting for certificate rotation")
				time.Sleep(time.Minute)
			}
		}

		/*
			if window.Contains(time.Now()) && rebootRequired() {
				node, err := client.CoreV1().Nodes().Get(nodeID, metav1.GetOptions{})
				if err != nil {
					logrus.Fatal(err)
				}
				_, exist := node.GetLabels()["node-role.kubernetes.io/master="]
				if !exist {
					logrus.Infof("Skipping worker nodes")
					continue
				}

				nodeMeta.Unschedulable = node.Spec.Unschedulable

				if acquire(lock, &nodeMeta) {
					rotateCertificate(nodeID)
					for {
						logrus.Infof("Waiting for certificate rotation")
						time.Sleep(time.Minute)
					}
				}
			}
		*/
		<-periodChannel
	}
}

func rebootRequired() bool {
	return true
}

func rotateCertificate(nodeID string) {
	logrus.Infof("Commanding certificate rotate")

	// Relies on hostPID:true and privileged:true to enter host mount space
	if err := newCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/cp", "-rf", "/etc/kubernetes/pki", "/etc/kubernetes/pki.bak").Run(); err != nil {
		logrus.Fatalf("Error invoking cp command: %v", err)
	}
	if err := newCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/kubeadm", "alpha", "certs", "renew", "all").Run(); err != nil {
		logrus.Fatalf("Error invoking kubeadm command: %v", err)
	}
	if err := newCommand("/usr/bin/nsenter", "-m/proc/1/ns/mnt", "/usr/bin/systemctl", "restart", "kubelet").Run(); err != nil {
		logrus.Fatalf("Error invoking system command: %v", err)
	}
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
