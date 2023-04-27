package cragcast

import "time"

type Cragcast struct {
	Days []*DailyForecast
}

type DailyForecast struct {
	Date time.Time
	// these are the headlines for the day
	TotalPrecipMm       *float32
	ChanceOfPrecip      *float32
	LowOfDayFahrenheit  *float32
	HighOfDayFahrenheit *float32
	avgSkyCoverPercent  *float32
	HumidityHigh        *float32
	snowfallMm          *float32
	// hourly data that will be useful for graphing
	Hours []*HourlyForecast
}

type HourlyForecast struct {
	TotalPrecipMm         *float32
	ChanceOfPrecip        *float32
	TemperatureFahrenheit *float32
	skyCoverPercent       *float32
	Humidity              *float32
	snowfallMm            *float32
	windDegrees           *float32
	windKmH               *float32
	StartOfHour           time.Time
	EndOfHour             time.Time
}
