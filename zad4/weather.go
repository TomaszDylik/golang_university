package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WeatherStation trzyma prosty stan pogody i wysyla go dalej.
type WeatherStation struct {
	out chan<- WeatherData
}

// Broadcaster rozsyla jeden odczyt do kilku subskrybentow.
type Broadcaster struct {
	in          <-chan WeatherData
	subscribers []chan<- WeatherData
}

// WindFarm reprezentuje jedna farme wiatrowa.
type WindFarm struct {
	id            string
	weatherIn     <-chan WeatherData
	controlIn     <-chan CurtailmentCommand
	productionOut chan<- ProductionReport
	currentMW     float64
	limitMW       float64
}

// GenerateData robi kolejny, prosty krok pogody bez pelnej losowosci.
func (w *WeatherStation) GenerateData(previous WeatherData) WeatherData {
	step := previous.WeatherStep + 1
	wind := previous.WindSpeedKPH + float64(step%5-2)

	if wind < 6 {
		wind = 6
	}
	if wind > 18 {
		wind = 18
	}

	return WeatherData{
		WeatherStep:  step,
		WindSpeedKPH: wind,
	}
}

// Run uruchamia stacje pogodowa jako osobna gorutyne.
func (w *WeatherStation) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(WeatherStep)
		defer ticker.Stop()

		current := WeatherData{WindSpeedKPH: 10}

		for {
			select {
			case <-ctx.Done():
				fmt.Println("[WEATHER] Koniec pracy stacji pogodowej.")
				return
			case <-ticker.C:
				current = w.GenerateData(current)

				select {
				case w.out <- current:
				default:
				}

				if current.WeatherStep%WeatherPerGrid == 0 {
					fmt.Printf("[WEATHER] krok=%d wiatr=%.1f km/h\n", current.WeatherStep, current.WindSpeedKPH)
				}
			}
		}
	}()
}

// Run uruchamia broadcaster, ktory rozglasza pogode do odbiorcow.
func (b *Broadcaster) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("[BROADCASTER] Koniec rozglaszania danych.")
				return
			case weather := <-b.in:
				for _, ch := range b.subscribers {
					select {
					case ch <- weather:
					default:
					}
				}

				if weather.WeatherStep%WeatherPerGrid == 0 {
					fmt.Printf("[BROADCASTER] Rozeslano krok pogodowy %d.\n", weather.WeatherStep)
				}
			}
		}
	}()
}

// ID zwraca identyfikator farmy.
func (w *WindFarm) ID() string {
	return w.id
}

// CurrentOutputMW zwraca ostatnia policzona moc.
func (w *WindFarm) CurrentOutputMW() float64 {
	return w.currentMW
}

// SetCurtailment ustawia proste ograniczenie mocy farmy.
func (w *WindFarm) SetCurtailment(limitMW float64) {
	w.limitMW = limitMW
}

// Run uruchamia farme wiatrowa jako osobna gorutyne.
func (w *WindFarm) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("[WIND FARM] Koniec pracy farmy wiatrowej.")
				return
			case cmd := <-w.controlIn:
				w.SetCurtailment(cmd.LimitMW)
			case weather := <-w.weatherIn:
				power := weather.WindSpeedKPH * 0.6

				if w.limitMW > 0 && power > w.limitMW {
					power = w.limitMW
				}

				w.currentMW = power

				report := ProductionReport{
					SourceID:   w.id,
					CurrentMW:  power,
					GridStep:   (weather.WeatherStep-1)/WeatherPerGrid + 1,
					SourceKind: "wind",
				}

				select {
				case w.productionOut <- report:
				default:
				}

				if weather.WeatherStep%WeatherPerGrid == 0 {
					fmt.Printf("[WIND FARM] grid=%d moc=%.1f MW\n", report.GridStep, report.CurrentMW)
				}
			}
		}
	}()
}
