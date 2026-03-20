package main

import (
	"sync/atomic"
	"time"
)

type paymentSemaphore struct {
	ch              chan struct{}
	withdrawWaiters atomic.Int32
}

func newPaymentSemaphore() *paymentSemaphore {
	return &paymentSemaphore{ch: make(chan struct{}, 1)}
}

// acquireForWithdrawal blocks until the semaphore is acquired, signalling
// priority to any concurrent refund goroutines.
func (p *paymentSemaphore) acquireForWithdrawal() {
	p.withdrawWaiters.Add(1)
	p.ch <- struct{}{}
	p.withdrawWaiters.Add(-1)
}

// tryAcquireForRefund does a non-blocking acquire. It returns false if a
// withdrawal is waiting or the semaphore is already held.
func (p *paymentSemaphore) tryAcquireForRefund() bool {
	if p.withdrawWaiters.Load() > 0 {
		return false
	}
	select {
	case p.ch <- struct{}{}:
		// Double-check: a withdrawal may have started waiting between the Load
		// and the send above.
		if p.withdrawWaiters.Load() > 0 {
			<-p.ch
			return false
		}
		return true
	default:
		return false
	}
}

// releaseAfter releases the semaphore after d in a background goroutine so
// the caller can return immediately.
func (p *paymentSemaphore) releaseAfter(d time.Duration) {
	go func() {
		time.Sleep(d)
		<-p.ch
	}()
}
