package main

import (
	"context"
	"fmt"
	"sync"
)

// SimplePredictor zbiera pogode i robi prosta prognoze dla kolejnego etapu.
type SimplePredictor struct {
	weatherIn    <-chan WeatherData
	forecastOut  chan<- ForecastReport
	history      []WeatherData
	gridStep     int
	lastMW       float64
	currentTrend string
}

// BuildForecast buduje jedna prosta prognoze z ostatnich danych.
func (p *SimplePredictor) BuildForecast() ForecastReport {
	if len(p.history) == 0 {
		return ForecastReport{}
	}

	totalWind := 0.0
	for _, item := range p.history {
		totalWind += item.WindSpeedKPH
	}

	avgWind := totalWind / float64(len(p.history))
	predictedMW := avgWind * 0.6
	changePercent := 0.0

	if p.lastMW > 0 {
		changePercent = ((predictedMW - p.lastMW) / p.lastMW) * 100
	}

	if changePercent > 5 {
		p.currentTrend = "wzrost"
	} else if changePercent < -5 {
		p.currentTrend = "spadek"
	} else {
		p.currentTrend = "stabilnie"
	}

	p.lastMW = predictedMW

	return ForecastReport{
		GridStep:              p.gridStep,
		Horizon:               ForecastHorizon,
		ExpectedMW:            predictedMW,
		ExpectedChangePercent: changePercent,
		Summary:               p.currentTrend,
	}
}

// Run uruchamia predictora i wysyla jedna prognoze po 12 krokach pogody.
func (p *SimplePredictor) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("[PREDICTOR] Koniec pracy predictora.")
				return
			case weather := <-p.weatherIn:
				p.history = append(p.history, weather)

				if len(p.history) > PredictorBufferSize {
					p.history = p.history[1:]
				}

				if len(p.history) == PredictorBufferSize {
					p.gridStep++
					forecast := p.BuildForecast()

					select {
					case p.forecastOut <- forecast:
					default:
					}

					p.history = p.history[:0]
				}
			}
		}
	}()
}

// startForecastPreview tylko wypisuje prognozy, zanim powstanie GridHub.
func startForecastPreview(ctx context.Context, wg *sync.WaitGroup, in <-chan ForecastReport) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("[FORECAST] Koniec podgladu prognoz.")
				return
			case forecast := <-in:
				fmt.Printf(
					"[FORECAST] grid=%d prognoza=%.1f MW zmiana=%.1f%% trend=%s\n",
					forecast.GridStep,
					forecast.ExpectedMW,
					forecast.ExpectedChangePercent,
					forecast.Summary,
				)
			}
		}
	}()
}
