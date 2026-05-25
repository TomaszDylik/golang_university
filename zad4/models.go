package main

// DemandReport jest wysyłany przez konsumenta do GridHub wspólnym kanałem Fan-In.
type DemandReport struct {
	ConsumerID  string
	RequestedMW float64
	Priority    ConsumerPriority
	GridStep    int
	ReplyChan   chan<- SupplyStatus
}

// SupplyStatus jest odpowiedzią GridHub dla pojedynczego konsumenta.
type SupplyStatus struct {
	ConsumerID  string
	AllocatedMW float64
	LoadShed    bool
	Reason      string
}

// ForecastReport przenosi wynik pracy Predictora do GridHub.
type ForecastReport struct {
	GridStep              int
	Horizon               int
	ExpectedMW            float64
	ExpectedChangePercent float64
	Summary               string
}

// WeatherData reprezentuje jeden odczyt pogodowy w szybkiej skali czasu.
type WeatherData struct {
	WeatherStep  int
	WindSpeedKPH float64
}

// ProductionReport przenosi bieżącą produkcję źródła do GridHub.
type ProductionReport struct {
	SourceID   string
	CurrentMW  float64
	GridStep   int
	SourceKind string
}

// ConsumerRegistration pozwala dodać nowego konsumenta bez restartu systemu.
type ConsumerRegistration struct {
	ConsumerID string
	Priority   ConsumerPriority
	ReplyChan  chan SupplyStatus
}

// ESSCommand i ESSStatus są kontraktem między GridHub a magazynem energii.
type ESSCommand struct {
	Mode    string
	PowerMW float64
}

type ESSStatus struct {
	SoC     float64
	PowerMW float64
	Limited bool
}

// PlantCommand i PlantStatus są kontraktem dla elektrowni konwencjonalnej.
type PlantCommand struct {
	Action string
}

type PlantStatus struct {
	State       string
	AvailableMW float64
}

// CurtailmentCommand pozwala ograniczyć moc OZE przy nadwyżce.
type CurtailmentCommand struct {
	LimitMW float64
	Reason  string
}

// LogEntry jest minimalną strukturą pod zapis CSV albo JSON.
type LogEntry struct {
	GridStep  int
	Component string
	Event     string
	Message   string
	ValueMW   float64
	ValueSoC  float64
	LoadShed  bool
}
