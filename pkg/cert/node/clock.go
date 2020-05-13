package node

import "time"

type Clock interface {
	Now() time.Time
}

type RealClock struct {
}

func NewRealClock() Clock {
	return &RealClock{}
}

func (r *RealClock) Now() time.Time {
	return time.Now()
}
