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

package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	capi "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/weaveworks/kured/pkg/daemonsetlock"

	"github.com/jenting/kucero/controllers"
	"github.com/jenting/kucero/pkg/host"
	"github.com/jenting/kucero/pkg/pki/cert"
	"github.com/jenting/kucero/pkg/pki/signer"
)

var (
	version = "unreleased"

	// Command line flags
	apiServerHost, kubeconfig                   string
	pollingPeriod, expiryTimeToRotate, duration time.Duration
	dsNamespace, dsName, lockAnnotation         string
	enableKubeletCSRController                  bool
	metricsAddr                                 string
	leaderElectionID                            string
	caCertPath, caKeyPath                       string

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

	// general
	rootCmd.PersistentFlags().StringVar(&apiServerHost, "master", "",
		"Optional apiserver host address to connect to")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "",
		"Paths to a kubeconfig. Only required if out-of-cluster.")

	// kubeadm
	rootCmd.PersistentFlags().DurationVar(&pollingPeriod, "polling-period", time.Hour,
		"Certificate rotation check period")
	rootCmd.PersistentFlags().DurationVar(&expiryTimeToRotate, "renew-before", time.Hour*24*30,
		"Rotates certificate if certificate not after is below")
	rootCmd.PersistentFlags().StringVar(&dsNamespace, "ds-namespace", "kube-system",
		"The namespace containing daemonset on which to place lock")
	rootCmd.PersistentFlags().StringVar(&dsName, "ds-name", "kucero",
		"The name of daemonset on which to place lock")
	rootCmd.PersistentFlags().StringVar(&lockAnnotation, "lock-annotation", "caasp.suse.com/kucero-node-lock",
		"The annotation in which to record locking node")

	// kubelet CSR controller
	rootCmd.PersistentFlags().BoolVar(&enableKubeletCSRController, "enable-kubelet-csr-controller", true,
		"Enable kubelet CSR controller")
	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metrics-addr", ":8080",
		"The address the metric endpoint binds to")
	rootCmd.PersistentFlags().StringVar(&leaderElectionID, "leader-election-id", "kucero-leader-election",
		"The name of the configmap used to coordinate leader election between kucero-controllers")
	rootCmd.PersistentFlags().StringVar(&caCertPath, "ca-cert-path", "/etc/kubernetes/pki/ca.crt",
		"To sign CSR with this certificate file")
	rootCmd.PersistentFlags().StringVar(&caKeyPath, "ca-key-path", "/etc/kubernetes/pki/ca.key",
		"To sign CSR with this private key file")
	rootCmd.PersistentFlags().DurationVar(&duration, "duration", time.Hour*24*365,
		"Kubelet certificate duration")

	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
	}
}

func root(cmd *cobra.Command, args []string) {
	logrus.Infof("KUbernetes CErtificate ROtation Daemon: %s", version)

	nodeName := os.Getenv("KUCERO_NODE_NAME")
	if nodeName == "" {
		logrus.Fatal("KUCERO_NODE_NAME environment variable required")
	}

	// shifting certificate check polling period
	rand.Seed(time.Now().UnixNano())
	extra := rand.Intn(int(pollingPeriod.Seconds()))
	pollingPeriod = pollingPeriod + time.Duration(extra)*time.Second

	logrus.Infof("Node Name: %s", nodeName)
	logrus.Infof("Lock Annotation: %s/%s:%s", dsNamespace, dsName, lockAnnotation)
	logrus.Infof("Shifted Certificate Check Polling Period %v", pollingPeriod)
	logrus.Infof("Rotates Certificate If Expiry Time Less Than %v", expiryTimeToRotate)
	if enableKubeletCSRController {
		logrus.Infof("Leader election ID: %s", leaderElectionID)
		logrus.Infof("CA cert: %s", caCertPath)
		logrus.Infof("CA key: %s", caKeyPath)
	}

	rotateCertificateWhenNeeded(nodeName)
}

// nodeMeta is used to remember information across nodes
// whom is doing certificate rotation
type nodeMeta struct {
	Unschedulable bool `json:"unschedulable"`
}

func rotateCertificateWhenNeeded(nodeName string) {
	config, err := clientcmd.BuildConfigFromFlags(apiServerHost, kubeconfig)
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
		logrus.Fatal("Kucero supports running on master node only")
	}

	certNode := cert.NewNode(isMasterNode, nodeName, expiryTimeToRotate)

	lock := daemonsetlock.New(client, nodeName, dsNamespace, dsName, lockAnnotation)
	nodeMeta := nodeMeta{}
	if holding(lock, &nodeMeta) {
		release(lock)
	}

	if enableKubeletCSRController {
		go func() {
			mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
				Scheme:                  scheme,
				MetricsBindAddress:      metricsAddr,
				Port:                    9443,
				LeaderElection:          true,
				LeaderElectionNamespace: dsNamespace,
				LeaderElectionID:        leaderElectionID,
			})
			if err != nil {
				logrus.Fatal(err)
			}

			signer, err := signer.NewSigner(caCertPath, caKeyPath, duration)
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

			logrus.Info("Starting manager")
			if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
				logrus.Fatal(err)
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ch := time.Tick(pollingPeriod)
	for {
		select {
		case <-quit:
			logrus.Info("Quitting")
			return
		case <-ch:
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
				logrus.Infof("The expiry certificiates are %v", expiryCerts)

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
}
