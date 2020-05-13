package cert

import (
	"time"

	"github.com/jenting/kucero/pkg/cert/node"
)

// NewNode checks it is master node or worker node
// then returns the corresponding node certificate interface
func NewNode(isMasterNode bool, nodeName string, expiryTimeToRotate time.Duration) node.Certificate {
	if isMasterNode {
		return node.NewMaster(nodeName, expiryTimeToRotate)
	}
	return node.NewWorker(nodeName, expiryTimeToRotate)
}
