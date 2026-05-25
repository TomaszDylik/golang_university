package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SimplePlant reprezentuje prosta elektrownie konwencjonalna.
type SimplePlant struct {
	commandIn     <-chan PlantCommand
	statusOut     chan<- PlantStatus
	state         string
	warmUpCounter int
}

func (p *SimplePlant) sendStatus() {
	available := 0.0
	if p.state == "On" {
		available = PlantMaxPowerMW
	}

	status := PlantStatus{
		State:       p.state,
		AvailableMW: available,
	}

	select {
	case p.statusOut <- status:
	default:
	}
}

// Run uruchamia prosta elektrownie z etapem WarmUp.
func (p *SimplePlant) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(GridStep)
		defer ticker.Stop()

		if p.state == "" {
			p.state = "Off"
		}
		p.sendStatus()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("[PLANT] Koniec pracy elektrowni.")
				return
			case cmd := <-p.commandIn:
				switch cmd.Action {
				case "start":
					if p.state == "Off" {
						p.state = "WarmUp"
						p.warmUpCounter = PlantWarmUpSteps
						fmt.Println("[PLANT] Start. Przejscie do WarmUp.")
						p.sendStatus()
					}
				case "stop":
					if p.state != "Off" {
						p.state = "Off"
						p.warmUpCounter = 0
						fmt.Println("[PLANT] Stop. Stan Off.")
						p.sendStatus()
					}
				}
			case <-ticker.C:
				if p.state == "WarmUp" {
					p.warmUpCounter--
					if p.warmUpCounter <= 0 {
						p.state = "On"
						fmt.Printf("[PLANT] WarmUp zakonczony. Dostepna moc %.1f MW.\n", PlantMaxPowerMW)
						p.sendStatus()
					}
				}
			}
		}
	}()
}
