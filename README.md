![CI](https://github.com/jenting/kucero/workflows/CI/badge.svg)
![CD](https://github.com/jenting/kucero/workflows/CD/badge.svg)

![kucero](logo.png)

## Introduction

Kucero (KUbernetes CErtificate ROtation) is a Kubernetes daemonset that
performs _automatic_ Kubernetes control plane certificate rotation.

Kucero takes care both:
- kubeadm-managed certificates and kubeconfigs: kucero periodically watches the kubeadm generated certificates and kubeconfigs on host system, and renews certificates/kubeconfigs when the certificates/kubeconfigs residual time is below than user configured time period.
- kubelet server CSR: kucero controller watches kubelet server CSR, and then auto signs and approves kubelet server certificates with user-specified CA cert/key pair.

## Kubelet Configuration

To enable kubelet server TLS bootstrapping, you need to set `serverTLSBootstrap: true` in kubelet configuration file `/var/lib/kubelet/config.yaml`. This will enable kubelet to generates kubelet server CSR.

## Build Requirements

- Golang >= 1.13
- Docker
- Kustomize

## Kubeadm Compatibility

- kubeadm >= 1.15.0

## Installation

```
make docker-build IMG=<YOUR-DOCKER-REPOSITORY-IMAGE-NAME-TAG>
make docker-push IMG=<YOUR-DOCKER-REPOSITORY-IMAGE-NAME-TAG>
make deploy-manifest IMG=<YOUR-DOCKER-REPOSITORY-IMAGE-NAME-TAG>
```

## Configuration

The following arguments can be passed to kucero via the daemonset pod template:

```
Flags:
      --ca-cert-path string         sign CSR with this certificate file (default "/etc/kubernetes/pki/ca.crt")
      --ca-key-path string          sign CSR with this private key file (default "/etc/kubernetes/pki/ca.key")
      --ds-name string              name of daemonset on which to place lock (default "kucero")
      --ds-namespace string         namespace containing daemonset on which to place lock (default "kube-system")
      --enable-kucero-controller    enable kucero controller (default true)
  -h, --help                        help for kucero
      --leader-election-id string   the name of the configmap used to coordinate leader election between kucero-controllers (default "kucero-leader-election")
      --lock-annotation string      annotation in which to record locking node (default "caasp.suse.com/kucero-node-lock")
      --metrics-addr string         the address the metric endpoint binds to (default ":8080")
      --polling-period duration     certificate rotation check period (default 1h0m0s)
      --renew-before duration       rotates certificate before expiry is below (default 720h0m0s)
```

##

## Demo

- kubeadm
  [![asciicast](https://asciinema.org/a/340053.svg)](https://asciinema.org/a/340053)
- kubelet
  [![asciicast](https://asciinema.org/a/340054.svg)](https://asciinema.org/a/340054)
