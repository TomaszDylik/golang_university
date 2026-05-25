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
	return 2 + float64(gridStep%3)
}

// Run uruchamia konsumenta i wysyla DemandReport co GridStep.
func (c *SimpleConsumer) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(GridStep)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("[CONSUMER] Koniec pracy konsumenta.")
				return
			case status := <-c.replyChan:
				fmt.Printf("[CONSUMER] przydzial=%.1f MW powod=%s\n", status.AllocatedMW, status.Reason)
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

				fmt.Printf("[CONSUMER] step=%d popyt=%.1f MW\n", c.gridStep, demand)
			}
		}
	}()
}
