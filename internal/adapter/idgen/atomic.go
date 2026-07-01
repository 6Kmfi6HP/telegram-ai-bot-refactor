package idgen

import "sync/atomic"

// AtomicDraftGenerator generates monotonically increasing Telegram draft IDs.
type AtomicDraftGenerator struct {
	counter int64
}

func NewAtomicDraftGenerator() *AtomicDraftGenerator {
	return &AtomicDraftGenerator{}
}

func (g *AtomicDraftGenerator) Next() int {
	return int(atomic.AddInt64(&g.counter, 1))
}
