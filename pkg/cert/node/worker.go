package node

type Worker struct {
	nodeName string
}

func NewWorker(nodeName string) Certificate {
	return &Worker{
		nodeName: nodeName,
	}
}

func (w *Worker) CheckExpiration() error {
	// TODO: check worker node's kubelet server certificate
	return nil
}

func (w *Worker) Rotate() error {
	// TODO: rotate worker node's kubelet server certificate
	return nil
}
