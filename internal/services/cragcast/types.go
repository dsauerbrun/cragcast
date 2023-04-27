package cragcast

import "time"

type Cragcast struct {
	Days []*DailyForecast
}

type DailyForecast struct {
	Date time.Time
	// these are the headlines for the day
	TotalPrecip         float64
	ChanceOfPrecip      float64
	LowOfDayFahrenheit  float64
	HighOfDayFahrenheit float64
	Cloudiness          float64
	HumidityHigh        float64
	// hourly data that will be useful for graphing
	Hours []*HourlyForecast
}

type HourlyForecast struct {
	TotalPrecip           float64
	ChanceOfPrecip        float64
	TemperatureFahrenheit float64
	Cloudiness            float64
	Humidity              float64
	StartOfHour           time.Time
	EndOfHour             time.Time
}
