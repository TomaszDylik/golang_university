package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// GridHub zbiera dane z kanalow i liczy bilans sieci z load sheddingiem.
type GridHub struct {
	productionIn  <-chan ProductionReport
	forecastIn    <-chan ForecastReport
	demandIn      <-chan DemandReport
	logOut        chan<- LogEntry
	latestProd    ProductionReport
	latestFc      ForecastReport
	pendingDemand map[string]DemandReport
	gridStep      int
}

// allocate liczy przydział dla każdego konsumenta i odcina najniższe priorytety przy niedoborze.
func (g *GridHub) allocate() {
	if len(g.pendingDemand) == 0 {
		return
	}

	// Zbierz raporty i posortuj od najwyzszego priorytetu (1=Critical) do najnizszego (3=Residential).
	reports := make([]DemandReport, 0, len(g.pendingDemand))
	for _, r := range g.pendingDemand {
		reports = append(reports, r)
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Priority < reports[j].Priority
	})

	available := g.latestProd.CurrentMW

	for _, r := range reports {
		var status SupplyStatus
		if available >= r.RequestedMW {
			available -= r.RequestedMW
			status = SupplyStatus{
				ConsumerID:  r.ConsumerID,
				AllocatedMW: r.RequestedMW,
				LoadShed:    false,
				Reason:      "pelne zasilanie",
			}
		} else {
			status = SupplyStatus{
				ConsumerID:  r.ConsumerID,
				AllocatedMW: 0,
				LoadShed:    true,
				Reason:      "load shedding",
			}
			fmt.Printf("[GRID] LOAD SHED consumer=%s priorytet=%d\n", r.ConsumerID, r.Priority)
			if g.logOut != nil {
				select {
				case g.logOut <- LogEntry{
					GridStep:  g.gridStep,
					Component: "GridHub",
					Event:     "load_shed",
					Message:   r.ConsumerID,
					LoadShed:  true,
				}:
				default:
				}
			}
		}

		select {
		case r.ReplyChan <- status:
		default:
		}
	}

	g.pendingDemand = make(map[string]DemandReport)
}

// Run uruchamia glowna petle select GridHub.
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
				g.pendingDemand[demand.ConsumerID] = demand
			case <-ticker.C:
				g.gridStep++
				totalDemand := 0.0
				for _, r := range g.pendingDemand {
					totalDemand += r.RequestedMW
				}

				balance := g.latestProd.CurrentMW - totalDemand

				fmt.Printf(
					"[GRID] step=%d produkcja=%.1f MW popyt=%.1f MW prognoza=%.1f MW bilans=%.1f MW\n",
					g.gridStep,
					g.latestProd.CurrentMW,
					totalDemand,
					g.latestFc.ExpectedMW,
					balance,
				)

				if g.logOut != nil {
					select {
					case g.logOut <- LogEntry{
						GridStep:  g.gridStep,
						Component: "GridHub",
						Event:     "balance",
						ValueMW:   balance,
					}:
					default:
					}
				}

				g.allocate()
			}
		}
	}()
}
