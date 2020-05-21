package main

import (
	"math/rand"
	"os"
	"time"

	capi "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/weaveworks/kured/pkg/daemonsetlock"
	"github.com/weaveworks/kured/pkg/delaytick"

	"github.com/jenting/kucero/controllers"
	"github.com/jenting/kucero/pkg/pki/cert"
	"github.com/jenting/kucero/pkg/host"
	"github.com/jenting/kucero/pkg/pki/signer"
)

var (
	version = "unreleased"

	// Command line flags
	pollingPeriod, expiryTimeToRotate   time.Duration
	dsNamespace, dsName, lockAnnotation string
	metricsAddr                         string
	leaderElectionID                    string
	caCertPath, caKeyPath               string

	scheme = runtime.NewScheme()
)

func init() {
	_ = capi.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "kucero",
		Short: "KUbernetes CErtificate ROtation",
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

	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metrics-addr", ":8080",
		"The address the metric endpoint binds to")
	rootCmd.PersistentFlags().StringVar(&leaderElectionID, "leader-election-id", "kucero-leader-election",
		"The name of the configmap used to coordinate leader election between kucero-controllers")
	rootCmd.PersistentFlags().StringVar(&caCertPath, "ca-cert-path", "/etc/kubernetes/pki/ca.crt",
		"Sign CSR with this certificate file")
	rootCmd.PersistentFlags().StringVar(&caKeyPath, "ca-key-path", "/etc/kubernetes/pki/ca.key",
		"Sign CSR with this private key file")

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

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     true,
		LeaderElectionID:   leaderElectionID,
	})
	if err != nil {
		logrus.Fatal(err)
	}

	signer, err := signer.NewSigner(caCertPath, caKeyPath)
	if err != nil {
		logrus.Fatal(err)
	}

	if err := (&controllers.CertificateSigningRequestSigningReconciler{
		Client:        mgr.GetClient(),
		ClientSet:     k8sclient.NewForConfigOrDie(mgr.GetConfig()),
		Scheme:        mgr.GetScheme(),
		Signer:        signer,
		EventRecorder: mgr.GetEventRecorderFor("CSRSigningReconciler"),
	}).SetupWithManager(mgr); err != nil {
		logrus.Fatal(err)
	}
	// +kubebuilder:scaffold:builder

	logrus.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logrus.Fatal(err)
	}

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
