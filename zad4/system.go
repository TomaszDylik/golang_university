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

// startStarterSystem uruchamia obecne komponenty
func startStarterSystem(ctx context.Context, wg *sync.WaitGroup) {
	channels := newSystemChannels()
	criticalReply := make(chan SupplyStatus, 1)
	industrialReply := make(chan SupplyStatus, 1)
	residentialReply := make(chan SupplyStatus, 1)

	station := &WeatherStation{out: channels.WeatherRaw}
	broadcaster := &Broadcaster{
		in: channels.WeatherRaw,
		subscribers: []chan<- WeatherData{
			channels.BroadcasterToPredictor,
			channels.BroadcasterToWindFarm,
		},
	}
	windFarm := &WindFarm{
		id:            "wind_farm_1",
		weatherIn:     channels.BroadcasterToWindFarm,
		controlIn:     channels.CurtailmentChan,
		productionOut: channels.ProductionChan,
	}
	predictor := &SimplePredictor{
		weatherIn:   channels.BroadcasterToPredictor,
		forecastOut: channels.ForecastChan,
		history:     make([]WeatherData, 0, PredictorBufferSize),
	}
	gridHub := &GridHub{
		productionIn:  channels.ProductionChan,
		forecastIn:    channels.ForecastChan,
		demandIn:      channels.DemandChan,
		logOut:        channels.LogChan,
		pendingDemand: make(map[string]DemandReport),
	}
	criticalConsumer := &SimpleConsumer{
		id:        "critical_1",
		priority:  PriorityCritical,
		demandOut: channels.DemandChan,
		replyChan: criticalReply,
	}
	industrialConsumer := &SimpleConsumer{
		id:        "industrial_1",
		priority:  PriorityIndustrial,
		demandOut: channels.DemandChan,
		replyChan: industrialReply,
	}
	residentialConsumer := &SimpleConsumer{
		id:        "residential_1",
		priority:  PriorityResidential,
		demandOut: channels.DemandChan,
		replyChan: residentialReply,
	}

	dataLogger := &FileDataLogger{
		logIn:    channels.LogChan,
		filename: "grid_log.jsonl",
	}

	fmt.Println("[SYSTEM] Etap 8: DataLogger zapisuje zdarzenia do grid_log.jsonl")

	dataLogger.Run(ctx, wg)
	station.Run(ctx, wg)
	broadcaster.Run(ctx, wg)
	windFarm.Run(ctx, wg)
	predictor.Run(ctx, wg)
	gridHub.Run(ctx, wg)
	criticalConsumer.Run(ctx, wg)
	industrialConsumer.Run(ctx, wg)
	residentialConsumer.Run(ctx, wg)
}
