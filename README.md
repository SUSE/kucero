![Build Status](https://github.com/jenting/kucero/workflows/Build%20Status/badge.svg)

```
 _
| | ___   _  ___ ___ _ __ ___
| |/ / | | |/ __/ _ \ '__/ _ \
|   <| |_| | (_|  __/ | | (_) |
|_|\_\\__,_|\___\___|_|  \___/
```

## Introduction

Kucero (KUbernetes CErtificate ROtation) is a Kubernetes daemonset that
performs safe automatic Kubernetes control plane certificate rotation
when the certificate residual time is below than user configured time period.

## Requirements

Golang >= 1.13

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

## Demo

[![asciicast](https://asciinema.org/a/331662.svg)](https://asciinema.org/a/331662)
