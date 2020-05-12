package cert

import (
	"time"

	"github.com/jenting/kucero/pkg/cert/node"
)

func NewNode(isMasterNode bool, nodeName string, expiryTimeToRotate time.Duration) node.Certificate {
	if isMasterNode {
		return node.NewMaster(nodeName, expiryTimeToRotate)
	}
	return node.NewWorker(nodeName, expiryTimeToRotate)
}
