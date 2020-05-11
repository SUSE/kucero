package main

import (
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/weaveworks/kured/pkg/daemonsetlock"

	cert "github.com/jenting/kucero/pkg/cert"
)

var (
	version = "unreleased"

	// Command line flags
	pollingPeriod, expiryTimeToRotate   time.Duration
	dsNamespace, dsName, lockAnnotation string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kucero",
		Short: "KUbeadm CErtificate ROtation",
		Run:   root,
	}

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

// nodeMeta is used to remember information across rotate certificates
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

	node, err := client.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		logrus.Fatal(err)
	}

	isMasterNode := false
	_, exist := node.GetLabels()["node-role.kubernetes.io/master"]
	if exist {
		isMasterNode = true
	}

	certNode := cert.NewNode(isMasterNode, nodeName)

	lock := daemonsetlock.New(client, nodeName, dsNamespace, dsName, lockAnnotation)

	nodeMeta := nodeMeta{}
	if holding(lock, &nodeMeta) {
		if !nodeMeta.Unschedulable {
			// uncordon(nodeID)
		}
		release(lock)
	}

	timer := time.NewTimer(pollingPeriod)
	for {
		logrus.Info("Check certificate expiration")

		// check the certificate needs expiration
		if err := certNode.CheckExpiration(); err != nil {
			logrus.Fatal(err)
		}

		// certificates need to rotation
		nodeMeta.Unschedulable = node.Spec.Unschedulable

		if acquire(lock, &nodeMeta) {
			if !nodeMeta.Unschedulable {
				// drain(nodeID)
			}

			logrus.Info("Waiting for certificate rotation")

			if err := certNode.Rotate(); err != nil {
				logrus.Fatal(err)
			}

			logrus.Info("Certificate rotation done")
		}

		<-timer.C
	}
}
