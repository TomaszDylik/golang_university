package main

import "time"

// WeatherStep określa, jak często generowane są dane pogodowe
const WeatherStep = 5 * time.Millisecond

// GridStep określa, jak często GridHub podejmuje decyzje o alokacji energii.
const GridStep = 100 * time.Millisecond

// W ciągu jednego kroku GridStep mieści się dokładnie 12 kroków pogodowych.
const WeatherPerGrid = int(GridStep / WeatherStep)

// ForecastHorizon określa, na ile kroków sieciowych do przodu patrzy prognoza.
const ForecastHorizon = 5

// PredictorBufferSize przechowuje jedną godzinę symulacji pogodowej.
const PredictorBufferSize = WeatherPerGrid

type ConsumerPriority int

const (
	PriorityCritical ConsumerPriority = iota + 1
	PriorityIndustrial
	PriorityResidential
)
