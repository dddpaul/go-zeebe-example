package stats

import (
	"sync/atomic"
)

var (
	created  atomic.Int64
	approved atomic.Int64
	rejected atomic.Int64
)

type Stats struct {
	Created  int64 `json:"created"`
	Approved int64 `json:"approved"`
	Rejected int64 `json:"rejected"`
}

func IncrementCreated() {
	created.Add(1)
}

func IncrementApproved() {
	approved.Add(1)
}

func IncrementRejected() {
	rejected.Add(1)
}

func Get() Stats {
	return Stats{
		Created:  created.Load(),
		Approved: approved.Load(),
		Rejected: rejected.Load(),
	}
}
