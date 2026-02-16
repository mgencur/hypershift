//go:build e2ev2
// +build e2ev2

package backuprestore

import (
	"context"
	"log"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	"golang.org/x/sync/errgroup"
)

// Prober is the interface for a prober.
type Prober interface {
	// Stop terminates the prober.
	Stop()
}

type prober struct {
	waitGrp *errgroup.Group
	ctx     context.Context
	cancel  context.CancelFunc
}

// prober implements Prober
var _ Prober = (*prober)(nil)

// Stop implements Prober
func (p *prober) Stop() {
	// Stop all probing.
	p.cancel()

	_ = p.waitGrp.Wait()
}

// ProberManager is the interface for spawning probers.
type ProberManager interface {
	// The ProberManager should expose a way to collectively reason about spawned
	// probes as a sort of aggregating Prober.
	Prober

	// Spawn creates a new Prober. The function passed as argument will be
	// executed in a separate goroutine repeatedly until the context is cancelled.
	// It can call verification functions from the Gomega library such as Expect or Fail.
	Spawn(func()) Prober
}

type manager struct {
	m      sync.RWMutex
	probes []Prober
}

var _ ProberManager = (*manager)(nil)

// Spawn implements ProberManager. It spawns a new Prober, adds it to the manager pool and returns it.
func (m *manager) Spawn(f func()) Prober {
	m.m.Lock()
	defer m.m.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	errGrp, ctx := errgroup.WithContext(ctx)

	p := &prober{
		waitGrp: errGrp,
		ctx:     ctx,
		cancel:  cancel,
	}
	m.probes = append(m.probes, p)

	errGrp.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				func() {
					// GinkgoRecover() will recover from any panics and mark the test as failed.
					defer GinkgoRecover()
					f()
				}()
			}
		}
	})
	return p
}

// Stop implements ProberManager. It stops all probers in the manager pool.
func (m *manager) Stop() {
	m.m.Lock()
	defer m.m.Unlock()

	log.Println("Stopping all probers")

	errgrp := errgroup.Group{}
	for _, prober := range m.probes {
		errgrp.Go(func() error {
			prober.Stop()
			return nil
		})
	}
	_ = errgrp.Wait()
}

// NewProberManager creates a new manager for probes.
func NewProberManager() ProberManager {
	m := manager{
		probes: make([]Prober, 0),
	}
	return &m
}
