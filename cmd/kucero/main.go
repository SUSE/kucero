package main

import (
	"math/rand"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/weaveworks/kured/pkg/daemonsetlock"
	"github.com/weaveworks/kured/pkg/delaytick"

	"github.com/jenting/kucero/pkg/cert"
	"github.com/jenting/kucero/pkg/host"
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
	rootCmd.PersistentFlags().DurationVar(&expiryTimeToRotate, "renew-before", time.Hour*24*30,
		"rotates certificate before expiry is below")

	rootCmd.PersistentFlags().StringVar(&dsNamespace, "ds-namespace", "kube-system",
		"namespace containing daemonset on which to place lock")
	rootCmd.PersistentFlags().StringVar(&dsName, "ds-name", "kucero",
		"name of daemonset on which to place lock")
	rootCmd.PersistentFlags().StringVar(&lockAnnotation, "lock-annotation", "caasp.suse.com/kucero-node-lock",
		"annotation in which to record locking node")

	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
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

// nodeMeta is used to remember information across nodes
// whom is doing certificate rotation
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

	isMasterNode := true
	_, exist := node.GetLabels()["node-role.kubernetes.io/master"]
	if !exist {
		isMasterNode = false
		logrus.Fatalf("kucero supports running on master node only")
	}

	certNode := cert.NewNode(isMasterNode, nodeName, expiryTimeToRotate)

	lock := daemonsetlock.New(client, nodeName, dsNamespace, dsName, lockAnnotation)

	nodeMeta := nodeMeta{}
	if holding(lock, &nodeMeta) {
		release(lock)
	}

	source := rand.NewSource(time.Now().UnixNano())
	tick := delaytick.New(source, pollingPeriod)
	for range tick {
		logrus.Info("Check certificate expiration")

		// check the certificate needs expiration
		expiryCerts, err := certNode.CheckExpiration()
		if err != nil {
			logrus.Error(err)
		}

		// rotates the certificate if there are certificates going to expire
		// and the lock can be acquired.
		// if the lock cannot be acquired, it will wait `pollingPeriod` time
		// and try to acquire the lock again.
		if len(expiryCerts) > 0 && acquire(lock, &nodeMeta) {
			logrus.Infof("The expiry certificiates are %v\n", expiryCerts)

			if !nodeMeta.Unschedulable {
				host.Cordon(nodeName)
				host.Drain(nodeName)
			}

			logrus.Info("Waiting for certificate rotation")
			if err := certNode.Rotate(expiryCerts); err != nil {
				logrus.Error(err)
			}
			logrus.Info("Certificate rotation done")

			if !nodeMeta.Unschedulable {
				host.Uncordon(nodeName)
			}

			release(lock)
		}
	}
}
