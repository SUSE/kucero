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

package cert

import (
	"time"

	"github.com/jenting/kucero/pkg/pki/cert/node"
)

// NewNode checks it is master node or worker node
// then returns the corresponding node certificate interface
func NewNode(isMasterNode bool, nodeName string, expiryTimeToRotate time.Duration) node.Certificate {
	if !isMasterNode {
		return nil
	}

	return node.NewMaster(nodeName, expiryTimeToRotate)
}
