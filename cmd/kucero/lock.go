package main

import (
	"github.com/sirupsen/logrus"
	"github.com/weaveworks/kured/pkg/daemonsetlock"
)

func holding(lock *daemonsetlock.DaemonSetLock, metadata interface{}) bool {
	holding, err := lock.Test(metadata)
	if err != nil {
		logrus.Fatalf("Error testing lock: %v", err)
	}
	if holding {
		logrus.Info("Holding lock")
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
		logrus.Info("Acquired kucero lock")
		return true
	}
}

func release(lock *daemonsetlock.DaemonSetLock) {
	logrus.Info("Releasing lock")
	if err := lock.Release(); err != nil {
		logrus.Fatalf("Error releasing lock: %v", err)
	}
}
