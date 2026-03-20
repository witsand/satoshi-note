package main

import (
	"sync"
	"sync/atomic"
	"time"
)

type paymentSemaphore struct {
	ch              chan struct{}
	withdrawWaiters atomic.Int32
	relMu           sync.Mutex
	released        chan struct{}
}

func newPaymentSemaphore() *paymentSemaphore {
	return &paymentSemaphore{
		ch:       make(chan struct{}, 1),
		released: make(chan struct{}),
	}
}

func (p *paymentSemaphore) getReleasedCh() chan struct{} {
	p.relMu.Lock()
	defer p.relMu.Unlock()
	return p.released
}

func (p *paymentSemaphore) doRelease() {
	p.relMu.Lock()
	oldCh := p.released
	p.released = make(chan struct{})
	p.relMu.Unlock()

	<-p.ch
	close(oldCh)
}

func (p *paymentSemaphore) releaseAfter(d time.Duration) {
	go func() {
		time.Sleep(d)
		p.doRelease()
	}()
}

// acquireForWithdrawal blocks up to 5 seconds. Returns false on timeout.
func (p *paymentSemaphore) acquireForWithdrawal() bool {
	p.withdrawWaiters.Add(1)
	defer p.withdrawWaiters.Add(-1)

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for {
		relCh := p.getReleasedCh()
		select {
		case p.ch <- struct{}{}:
			return true
		case <-timeout.C:
			return false
		case <-relCh:
			// released; retry
		}
	}
}

// acquireForRefund blocks indefinitely, yielding to any waiting withdrawal.
func (p *paymentSemaphore) acquireForRefund() {
	for {
		if p.withdrawWaiters.Load() > 0 {
			<-p.getReleasedCh()
			continue
		}
		relCh := p.getReleasedCh()
		select {
		case p.ch <- struct{}{}:
			if p.withdrawWaiters.Load() > 0 {
				p.doRelease() // yield, no cooldown since no payment was made
				continue
			}
			return
		default:
			<-relCh
		}
	}
}
