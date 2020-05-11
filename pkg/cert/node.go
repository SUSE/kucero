package cert

import (
	"github.com/jenting/kucero/pkg/cert/node"
)

func NewNode(isMasterNode bool, nodeName string) node.Certificate {
	if isMasterNode {
		return node.NewMaster(nodeName)
	}
	return node.NewWorker(nodeName)
}
