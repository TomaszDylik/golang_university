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
	productionIn    <-chan ProductionReport
	forecastIn      <-chan ForecastReport
	demandIn        <-chan DemandReport
	registrationIn  <-chan ConsumerRegistration
	essStatusIn     <-chan ESSStatus
	plantStatusIn   <-chan PlantStatus
	essCommandOut   chan<- ESSCommand
	plantCommandOut chan<- PlantCommand
	logOut          chan<- LogEntry
	latestProd      ProductionReport
	latestFc        ForecastReport
	latestESS       ESSStatus
	latestPlant     PlantStatus
	pendingDemand   map[string]DemandReport
	registered      map[string]ConsumerRegistration
	loadShedCount   int
	gridStep        int
}

func (g *GridHub) logEvent(event string, message string, valueMW float64, valueSoC float64, loadShed bool) {
	if g.logOut == nil {
		return
	}

	select {
	case g.logOut <- LogEntry{
		GridStep:  g.gridStep,
		Component: "GridHub",
		Event:     event,
		Message:   message,
		ValueMW:   valueMW,
		ValueSoC:  valueSoC,
		LoadShed:  loadShed,
	}:
	default:
	}
}

func minFloat(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func (g *GridHub) plantPotential() float64 {
	if g.latestPlant.State == "On" {
		return g.latestPlant.AvailableMW
	}
	return 0
}

func (g *GridHub) essPotential() float64 {
	available := g.latestESS.SoC * ESSCapacityMWh
	if available > ESSMaxPowerMW {
		return ESSMaxPowerMW
	}
	return available
}

func (g *GridHub) dispatch(totalDemand float64) float64 {
	baseAvailable := g.latestProd.CurrentMW + g.plantPotential()
	deficitAfterPlant := totalDemand - baseAvailable
	forecastLow := g.latestFc.ExpectedMW < totalDemand

	if (deficitAfterPlant > 0 || forecastLow) && g.latestPlant.State == "Off" {
		select {
		case g.plantCommandOut <- PlantCommand{Action: "start"}:
			fmt.Println("[GRID] Polecenie START dla elektrowni.")
			g.logEvent("plant_start", "forecast or deficit requires plant", 0, g.latestESS.SoC, false)
		default:
		}
	}

	plannedESS := 0.0
	if deficitAfterPlant > 0 {
		dischargeMW := minFloat(deficitAfterPlant, g.essPotential())
		if dischargeMW > 0 {
			select {
			case g.essCommandOut <- ESSCommand{Mode: "discharge", PowerMW: dischargeMW}:
				plannedESS = dischargeMW
				g.logEvent("ess_discharge", "supporting deficit", dischargeMW, g.latestESS.SoC, false)
			default:
			}
		}
	} else {
		if g.latestPlant.State == "On" && g.latestESS.SoC > 0.85 && g.latestFc.ExpectedMW >= totalDemand {
			select {
			case g.plantCommandOut <- PlantCommand{Action: "stop"}:
				fmt.Println("[GRID] Polecenie STOP dla elektrowni.")
				g.logEvent("plant_stop", "surplus detected", 0, g.latestESS.SoC, false)
			default:
			}
		}

		surplus := baseAvailable - totalDemand
		chargeMW := minFloat(surplus, ESSMaxPowerMW)
		if chargeMW > 0 && g.latestESS.SoC < 1.0 {
			select {
			case g.essCommandOut <- ESSCommand{Mode: "charge", PowerMW: chargeMW}:
				g.logEvent("ess_charge", "storing surplus", chargeMW, g.latestESS.SoC, false)
			default:
			}
		}
	}

	return baseAvailable + plannedESS
}

func (g *GridHub) printReport(totalDemand float64, balance float64, available float64) {
	activeConsumers := len(g.pendingDemand)
	message := fmt.Sprintf(
		"aktywni=%d produkcja=%.1fMW plant=%.1fMW ess_soc=%.2f popyt=%.1fMW prognoza=%.1fMW dostepne=%.1fMW bilans=%.1fMW",
		activeConsumers,
		g.latestProd.CurrentMW,
		g.plantPotential(),
		g.latestESS.SoC,
		totalDemand,
		g.latestFc.ExpectedMW,
		available,
		balance,
	)

	fmt.Printf("[REPORT] step=%d %s\n", g.gridStep, message)
	g.logEvent("report", message, balance, g.latestESS.SoC, false)
}

// allocate liczy przydział dla każdego konsumenta i odcina najniższe priorytety przy niedoborze.
func (g *GridHub) allocate(available float64) {
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
			g.loadShedCount++
			fmt.Printf("[GRID] LOAD SHED consumer=%s priorytet=%d\n", r.ConsumerID, r.Priority)
			g.logEvent("load_shed", r.ConsumerID, 0, g.latestESS.SoC, true)
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

		g.latestESS = ESSStatus{SoC: ESSInitialSoC}
		g.latestPlant = PlantStatus{State: "Off"}

		for {
			select {
			case <-ctx.Done():
				fmt.Printf("[GRID] Koniec pracy GridHub. load_shed=%d zarejestrowani=%d soc=%.2f plant=%s\n", g.loadShedCount, len(g.registered), g.latestESS.SoC, g.latestPlant.State)
				return
			case production := <-g.productionIn:
				g.latestProd = production
			case forecast := <-g.forecastIn:
				g.latestFc = forecast
			case ess := <-g.essStatusIn:
				g.latestESS = ess
			case plant := <-g.plantStatusIn:
				g.latestPlant = plant
			case reg := <-g.registrationIn:
				g.registered[reg.ConsumerID] = reg
				fmt.Printf("[GRID] Rejestracja konsumenta: id=%s priorytet=%d\n", reg.ConsumerID, reg.Priority)
				g.logEvent("consumer_registered", reg.ConsumerID, 0, g.latestESS.SoC, false)
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

				g.logEvent("balance", "", balance, g.latestESS.SoC, false)
				available := g.dispatch(totalDemand)
				finalBalance := available - totalDemand

				if g.gridStep%GridReportEvery == 0 {
					g.printReport(totalDemand, finalBalance, available)
				}

				g.allocate(available)
			}
		}
	}()
}
