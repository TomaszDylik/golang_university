package main

import (
	"context"
	"sync"
)

// definuje interfejsy dla kluczowych komponentów
type EnergySource interface {
	Run(ctx context.Context, wg *sync.WaitGroup)
	ID() string
	CurrentOutputMW() float64
	SetCurtailment(limitMW float64)
}

// komponent analizujący pogodę i budujący prognozy dla GridHub
type Predictor interface {
	Run(ctx context.Context, wg *sync.WaitGroup)
	BuildForecast() ForecastReport
}

// Consumer opisuje odbiorcę energii działającego jako osobna gorutyna
type Consumer interface {
	Run(ctx context.Context, wg *sync.WaitGroup)
	ID() string
	Priority() ConsumerPriority
	CalculateDemand(gridStep int) float64
}

// EnergyStorage opisuje magazyn energii ESS.
type EnergyStorage interface {
	Run(ctx context.Context, wg *sync.WaitGroup)
	CurrentSoC() float64
	Charge(powerMW float64) float64
	Discharge(powerMW float64) float64
}

// WeatherProvider opisuje źródło danych pogodowych.
type WeatherProvider interface {
	Run(ctx context.Context, wg *sync.WaitGroup)
	GenerateData(previous WeatherData) WeatherData
}

// DataLogger opisuje asynchroniczny komponent odpowiedzialny za trwały zapis danych.
type DataLogger interface {
	Run(ctx context.Context, wg *sync.WaitGroup)
	Log(entry LogEntry)
	Flush() error
}
