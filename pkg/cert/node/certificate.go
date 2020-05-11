package node

type Certificate interface {
	CheckExpiration() error
	Rotate() error
}
