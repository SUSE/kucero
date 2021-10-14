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
	"time"

	"github.com/sirupsen/logrus"
	"github.com/weaveworks/kured/pkg/daemonsetlock"
)

func holding(lock *daemonsetlock.DaemonSetLock, metadata interface{}) bool {
	holding, err := lock.Test(metadata)
	if err != nil {
		logrus.Errorf("Error testing lock: %v", err)
	}
	if holding {
		logrus.Info("Holding lock")
	}
	return holding
}

func acquire(lock *daemonsetlock.DaemonSetLock, metadata interface{}) bool {
	holding, holder, err := lock.Acquire(metadata, time.Minute)
	switch {
	case err != nil:
		logrus.Errorf("Error acquiring lock: %v", err)
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
		logrus.Errorf("Error releasing lock: %v", err)
	}
}
