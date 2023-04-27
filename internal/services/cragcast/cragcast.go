// this will contain functions for different forecast functionality.
// eg. GetForecast, GetHumidity, GetRainfall, etc...(not necessarily these, just giving them as examples to illustrate)

package cragcast

import (
	client "github.com/dsauerbrun/cragcast/pkg/weather-client"
)

func GetForecast(cragId int) (*Cragcast, error) {
	// TODO: get crag's lat/lng from DB based on cragId. For now, we use boulder which is 40.0294122,-105.3223779
	lat, lng := 40.0294122, -105.3223779
	weatherClientForecast, err := client.New().GetForecast(lat, lng)
	if err != nil {
		return nil, err
	}

	// what do people actually care about in a climbing forecast?
	// rainfall %, amount of rainfall
	// temps by the hour as well as the highest temp
	// cloudiness
	// humidity by the hour as well as highest humidity
	// TODO: People typically care about the duration from morning until night so we should adjust for location's timezone and only take into account daylight hours

	cragcast := &Cragcast{
		Days: []*DailyForecast{},
	}
	for _, period := range weatherClientForecast.Properties.Periods {
		cragcastDayIndex := len(cragcast.Days) - 1
		// do we need a timezone conversion here?
		day := period.StartTime.Day()
		if len(cragcast.Days) == 0 || cragcast.Days[cragcastDayIndex].Date.Day() != day {
			// we are on a new day so we append a day to our cragcast
			cragcast.Days = append(cragcast.Days, &DailyForecast{
				Date:               period.StartTime,
				LowOfDayFahrenheit: 1000,
				Hours:              []*HourlyForecast{},
			})
			cragcastDayIndex++
		}

		cragcastDay := cragcast.Days[cragcastDayIndex]
		// noaa api doesnt expose precip amount via api, probably hidden somewhere i cant find
		cragcastDay.TotalPrecip = 0
		if cragcastDay.ChanceOfPrecip < float64(period.ProbabilityOfPrecipitation.Value) {
			cragcastDay.ChanceOfPrecip = float64(period.ProbabilityOfPrecipitation.Value)
		}
		if cragcastDay.LowOfDayFahrenheit > float64(period.Temperature) {
			cragcastDay.LowOfDayFahrenheit = float64(period.Temperature)
		}
		if cragcastDay.HighOfDayFahrenheit < float64(period.Temperature) {
			cragcastDay.HighOfDayFahrenheit = float64(period.Temperature)
		}

		// cant find the sky cover % in their api, leaving it for now
		cragcastDay.Cloudiness = 0

		if cragcastDay.HumidityHigh < float64(period.RelativeHumidity.Value) {
			cragcastDay.HumidityHigh = float64(period.RelativeHumidity.Value)
		}

		cragcastDay.Hours = append(cragcastDay.Hours, &HourlyForecast{
			TotalPrecip:           0,
			ChanceOfPrecip:        float64(period.ProbabilityOfPrecipitation.Value),
			TemperatureFahrenheit: float64(period.Temperature),
			Cloudiness:            0,
			Humidity:              float64(period.RelativeHumidity.Value),
			StartOfHour:           period.StartTime,
			EndOfHour:             period.EndTime,
		})
	}

	return cragcast, nil
}
