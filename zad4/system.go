package main

import (
	"context"
	"fmt"
	"sync"
)

// SystemChannels zbiera w jednym miejscu wszystkie kanały z diagramu i polecenia.
// Dzięki temu już na etapie planu wiadomo, jak komponenty będą się komunikować.
type SystemChannels struct {
	WeatherRaw             chan WeatherData
	BroadcasterToPredictor chan WeatherData
	BroadcasterToWindFarm  chan WeatherData

	ProductionChan   chan ProductionReport
	ForecastChan     chan ForecastReport
	DemandChan       chan DemandReport
	RegistrationChan chan ConsumerRegistration

	ESSCommandChan   chan ESSCommand
	ESSStatusChan    chan ESSStatus
	PlantCommandChan chan PlantCommand
	PlantStatusChan  chan PlantStatus
	CurtailmentChan  chan CurtailmentCommand

	LogChan       chan LogEntry
	SupplyReplies map[string]chan SupplyStatus
}

func newSystemChannels() *SystemChannels {
	return &SystemChannels{
		WeatherRaw:             make(chan WeatherData, 1),
		BroadcasterToPredictor: make(chan WeatherData, WeatherPerGrid),
		BroadcasterToWindFarm:  make(chan WeatherData, WeatherPerGrid),

		ProductionChan:   make(chan ProductionReport, 8),
		ForecastChan:     make(chan ForecastReport, 1),
		DemandChan:       make(chan DemandReport, 32),
		RegistrationChan: make(chan ConsumerRegistration, 8),

		ESSCommandChan:   make(chan ESSCommand, 4),
		ESSStatusChan:    make(chan ESSStatus, 4),
		PlantCommandChan: make(chan PlantCommand, 4),
		PlantStatusChan:  make(chan PlantStatus, 4),
		CurtailmentChan:  make(chan CurtailmentCommand, 4),

		LogChan:       make(chan LogEntry, 64),
		SupplyReplies: make(map[string]chan SupplyStatus),
	}
}

// startStarterSystem nie uruchamia jeszcze logiki biznesowej.
// Ma tylko pokazać, jakie gorutyny i kanały będą spinane w dalszych etapach.
func startStarterSystem(ctx context.Context, wg *sync.WaitGroup) {
	channels := newSystemChannels()

	// Przykładowe kanały odpowiedzi dla startowych konsumentów.
	channels.SupplyReplies["critical_1"] = make(chan SupplyStatus, 1)
	channels.SupplyReplies["industrial_1"] = make(chan SupplyStatus, 1)
	channels.SupplyReplies["residential_1"] = make(chan SupplyStatus, 1)

	fmt.Println("[PLAN] Zainicjalizowano szkic architektury kanałów.")
	fmt.Println("[PLAN] Jeden typ OZE: farma wiatrowa.")
	fmt.Println("[PLAN] GridHub będzie centrum decyzji między pogodą, popytem, ESS i elektrownią.")

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		_ = channels
		fmt.Println("[PLAN] Graceful shutdown szkieletu zakończony.")
	}()
}
