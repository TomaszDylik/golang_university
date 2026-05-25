package main

import "time"

// WeatherStep określa, jak często generowane są dane pogodowe
const WeatherStep = 5 * time.Millisecond

// GridStep określa, jak często GridHub podejmuje decyzje o alokacji energii.
const GridStep = 100 * time.Millisecond

// W tym projekcie przyjmujemy 12 krokow pogodowych na 1 GridStep.
const WeatherPerGrid = 12

// ForecastHorizon określa, na ile kroków sieciowych do przodu patrzy prognoza.
const ForecastHorizon = 5

// PredictorBufferSize przechowuje jedną godzinę symulacji pogodowej.
const PredictorBufferSize = WeatherPerGrid

// GridReportEvery określa, co ile krokow GridStep wypisac raport stanu.
const GridReportEvery = 5

// Parametry prostego magazynu energii.
const ESSCapacityMWh = 12.0
const ESSMaxPowerMW = 4.0
const ESSInitialSoC = 0.5

// Parametry prostej elektrowni konwencjonalnej.
const PlantMaxPowerMW = 8.0
const PlantWarmUpSteps = 2

type ConsumerPriority int

const (
	PriorityCritical ConsumerPriority = iota + 1
	PriorityIndustrial
	PriorityResidential
)
