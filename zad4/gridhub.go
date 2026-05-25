package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// GridHub zbiera dane z kanalow i liczy prosty bilans sieci.
type GridHub struct {
	productionIn <-chan ProductionReport
	forecastIn   <-chan ForecastReport
	demandIn     <-chan DemandReport
	latestProd   ProductionReport
	latestFc     ForecastReport
	latestDemand DemandReport
	gridStep     int
}

// Run uruchamia glowna petle select dla obecnego etapu.
func (g *GridHub) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(GridStep)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("[GRID] Koniec pracy GridHub.")
				return
			case production := <-g.productionIn:
				g.latestProd = production
			case forecast := <-g.forecastIn:
				g.latestFc = forecast
			case demand := <-g.demandIn:
				g.latestDemand = demand

				status := SupplyStatus{
					ConsumerID:  demand.ConsumerID,
					AllocatedMW: demand.RequestedMW,
					LoadShed:    false,
					Reason:      "pelne zasilanie",
				}

				select {
				case demand.ReplyChan <- status:
				default:
				}
			case <-ticker.C:
				g.gridStep++
				balance := g.latestProd.CurrentMW - g.latestDemand.RequestedMW

				fmt.Printf(
					"[GRID] step=%d produkcja=%.1f MW popyt=%.1f MW prognoza=%.1f MW bilans=%.1f MW\n",
					g.gridStep,
					g.latestProd.CurrentMW,
					g.latestDemand.RequestedMW,
					g.latestFc.ExpectedMW,
					balance,
				)
			}
		}
	}()
}
