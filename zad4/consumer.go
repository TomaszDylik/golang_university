package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SimpleConsumer reprezentuje jednego prostego odbiorce energii.
type SimpleConsumer struct {
	id        string
	priority  ConsumerPriority
	demandOut chan<- DemandReport
	replyChan chan SupplyStatus
	gridStep  int
}

// ID zwraca identyfikator konsumenta.
func (c *SimpleConsumer) ID() string {
	return c.id
}

// Priority zwraca priorytet konsumenta.
func (c *SimpleConsumer) Priority() ConsumerPriority {
	return c.priority
}

// CalculateDemand liczy proste zapotrzebowanie dla obecnego kroku.
func (c *SimpleConsumer) CalculateDemand(gridStep int) float64 {
	switch c.priority {
	case PriorityCritical:
		return 4
	case PriorityIndustrial:
		if gridStep%2 == 0 {
			return 5.5
		}
		return 5
	default:
		return 2 + float64(gridStep%3)
	}
}

// Run uruchamia konsumenta i wysyla DemandReport co GridStep.
func (c *SimpleConsumer) Run(ctx context.Context, wg *sync.WaitGroup) {
	c.run(ctx, wg, false)
}

// RunReserved uruchamia konsumenta, zakladajac ze slot w WaitGroup jest juz zarezerwowany.
func (c *SimpleConsumer) RunReserved(ctx context.Context, wg *sync.WaitGroup) {
	c.run(ctx, wg, true)
}

func (c *SimpleConsumer) run(ctx context.Context, wg *sync.WaitGroup, reserved bool) {
	if !reserved {
		wg.Add(1)
	}

	go func() {
		defer wg.Done()

		ticker := time.NewTicker(GridStep)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Printf("[CONSUMER %s] Koniec pracy konsumenta.\n", c.id)
				return
			case status := <-c.replyChan:
				fmt.Printf("[CONSUMER %s] przydzial=%.1f MW powod=%s\n", c.id, status.AllocatedMW, status.Reason)
			case <-ticker.C:
				c.gridStep++
				demand := c.CalculateDemand(c.gridStep)

				report := DemandReport{
					ConsumerID:  c.id,
					RequestedMW: demand,
					Priority:    c.priority,
					GridStep:    c.gridStep,
					ReplyChan:   c.replyChan,
				}

				select {
				case c.demandOut <- report:
				default:
				}

				fmt.Printf("[CONSUMER %s] step=%d popyt=%.1f MW\n", c.id, c.gridStep, demand)
			}
		}
	}()
}
