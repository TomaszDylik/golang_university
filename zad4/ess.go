package main

import (
	"context"
	"fmt"
	"sync"
)

// SimpleESS reprezentuje prosty magazyn energii sterowany przez GridHub.
type SimpleESS struct {
	commandIn <-chan ESSCommand
	statusOut chan<- ESSStatus
	soc       float64
}

// CurrentSoC zwraca aktualny poziom naladowania.
func (e *SimpleESS) CurrentSoC() float64 {
	return e.soc
}

// Charge laduje magazyn z ograniczeniem pojemnosci i mocy.
func (e *SimpleESS) Charge(powerMW float64) float64 {
	if powerMW < 0 {
		powerMW = 0
	}
	if powerMW > ESSMaxPowerMW {
		powerMW = ESSMaxPowerMW
	}

	freeCapacity := (1.0 - e.soc) * ESSCapacityMWh
	actual := powerMW
	if actual > freeCapacity {
		actual = freeCapacity
	}

	if ESSCapacityMWh > 0 {
		e.soc += actual / ESSCapacityMWh
	}
	if e.soc > 1.0 {
		e.soc = 1.0
	}

	return actual
}

// Discharge oddaje energie do sieci z ograniczeniem pojemnosci i mocy.
func (e *SimpleESS) Discharge(powerMW float64) float64 {
	if powerMW < 0 {
		powerMW = 0
	}
	if powerMW > ESSMaxPowerMW {
		powerMW = ESSMaxPowerMW
	}

	availableEnergy := e.soc * ESSCapacityMWh
	actual := powerMW
	if actual > availableEnergy {
		actual = availableEnergy
	}

	if ESSCapacityMWh > 0 {
		e.soc -= actual / ESSCapacityMWh
	}
	if e.soc < 0 {
		e.soc = 0
	}

	return actual
}

func (e *SimpleESS) sendStatus(powerMW float64, limited bool) {
	status := ESSStatus{
		SoC:     e.soc,
		PowerMW: powerMW,
		Limited: limited,
	}

	select {
	case e.statusOut <- status:
	default:
	}
}

// Run uruchamia prosty worker ESS.
func (e *SimpleESS) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		e.sendStatus(0, false)

		for {
			select {
			case <-ctx.Done():
				fmt.Println("[ESS] Koniec pracy magazynu energii.")
				return
			case cmd := <-e.commandIn:
				switch cmd.Mode {
				case "charge":
					actual := e.Charge(cmd.PowerMW)
					limited := actual < cmd.PowerMW
					fmt.Printf("[ESS] ladowanie=%.1f MW soc=%.2f\n", actual, e.soc)
					e.sendStatus(-actual, limited)
				case "discharge":
					actual := e.Discharge(cmd.PowerMW)
					limited := actual < cmd.PowerMW
					fmt.Printf("[ESS] rozladowanie=%.1f MW soc=%.2f\n", actual, e.soc)
					e.sendStatus(actual, limited)
				default:
					e.sendStatus(0, false)
				}
			}
		}
	}()
}
