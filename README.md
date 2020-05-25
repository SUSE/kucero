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
when the certificate rediaul time is below than user configured time period.

## Requirements

Golang >= 1.13

## Installation

```
make docker-build DOCKER_IMAGE=<YOUR-DOCKER-REPOSITORY-IMAGE-NAME-TAG>
make docker-push DOCKER_IMAGE=<YOUR-DOCKER-REPOSITORY-IMAGE-NAME-TAG>
make deploy-manifest DOCKER_IMAGE=<YOUR-DOCKER-REPOSITORY-IMAGE-NAME-TAG>
```

## Configuration

The following arguments can be passed to kucero via the daemonset pod template:

```
Flags:
      --ds-name string            name of daemonset on which to place lock (default "kucero")
      --ds-namespace string       namespace containing daemonset on which to place lock (default "kube-system")
  -h, --help                      help for kucero
      --lock-annotation string    annotation in which to record locking node (default "caasp.suse.com/kucero-node-lock")
      --polling-period duration   certificate rotation check period (default 1h0m0s)
      --renew-before duration     rotates certificate before expiry is below (default 720h0m0s)
```

## Demo

[![asciicast](https://asciinema.org/a/331662.svg)](https://asciinema.org/a/331662)
