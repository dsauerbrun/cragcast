// this will contain functions for different forecast functionality.
// eg. GetForecast, GetHumidity, GetRainfall, etc...(not necessarily these, just giving them as examples to illustrate)

package cragcast

import (
	"errors"
	"sort"
	"strings"
	"time"

	client "github.com/dsauerbrun/cragcast/pkg/weather-client"
	noaaTypes "github.com/dsauerbrun/cragcast/pkg/weather-client/noaa"
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

	hourlyForecasts, err := processHours(weatherClientForecast)
	if err != nil {
		return nil, err
	}
	cragcast := &Cragcast{
		Days: []*DailyForecast{},
	}

	for _, hour := range hourlyForecasts {
		cragcastDayIndex := len(cragcast.Days) - 1
		// do we need a timezone conversion here?
		day := hour.StartOfHour.Day()
		if len(cragcast.Days) == 0 || cragcast.Days[cragcastDayIndex].Date.Day() != day {
			// we are on a new day so we append a day to our cragcast
			initialLowFahrenheit := float32(1000)
			cragcast.Days = append(cragcast.Days, &DailyForecast{
				Date:               hour.StartOfHour,
				LowOfDayFahrenheit: &initialLowFahrenheit,
				Hours:              []*HourlyForecast{},
			})
			cragcastDayIndex++
		}

		cragcastDay := cragcast.Days[cragcastDayIndex]
		if hour.TotalPrecipMm != nil {
			newPrecipMm := getPointerValueWithDefault(cragcastDay.TotalPrecipMm) + *hour.TotalPrecipMm
			cragcastDay.TotalPrecipMm = &newPrecipMm
		}

		if hour.ChanceOfPrecip != nil && getPointerValueWithDefault(cragcastDay.ChanceOfPrecip) < *hour.ChanceOfPrecip {
			cragcastDay.ChanceOfPrecip = hour.ChanceOfPrecip
		}
		if hour.TemperatureFahrenheit != nil && getPointerValueWithDefault(cragcastDay.LowOfDayFahrenheit) > *hour.TemperatureFahrenheit {
			cragcastDay.LowOfDayFahrenheit = hour.TemperatureFahrenheit
		}
		if hour.TemperatureFahrenheit != nil && getPointerValueWithDefault(cragcastDay.HighOfDayFahrenheit) < *hour.TemperatureFahrenheit {
			cragcastDay.HighOfDayFahrenheit = hour.TemperatureFahrenheit
		}

		if hour.Humidity != nil && getPointerValueWithDefault(cragcastDay.HumidityHigh) < *hour.Humidity {
			cragcastDay.HumidityHigh = hour.Humidity
		}

		if hour.snowfallMm != nil {
			newSnowfall := getPointerValueWithDefault(cragcastDay.snowfallMm) + *hour.snowfallMm
			cragcastDay.snowfallMm = &newSnowfall
		}

		cragcastDay.Hours = append(cragcastDay.Hours, hour)
	}

	return cragcast, nil
}

func processHours(weatherClientForecast *noaaTypes.GridpointGeoJson) ([]*HourlyForecast, error) {
	hours := []*HourlyForecast{}
	// make a dictionary where times are the keys and values are hourly forecasts, loop through all sky cover, temp, precip etc...
	// and add this info to each hour
	// once that is done, loop through keys to make the hourly forecast and then sort by date

	hoursMap := make(map[string]*HourlyForecast)

	quantValuesToProcess := []QuantitativeValueType{TEMPERATURE, HUMIDITY, PRECIPITATION_PROBABILITY, QUANTITATIVE_PRECIPITATION, SKY_COVER, SNOWFALL_AMOUNT, WIND_DIRECTION, WIND_SPEED}

	for _, quantValue := range quantValuesToProcess {
		err := processQuantitativeValues(quantValue, weatherClientForecast.Properties.AdditionalProperties, hoursMap)
		if err != nil {
			return nil, err
		}
	}

	for _, hourlyForecast := range hoursMap {
		hours = append(hours, hourlyForecast)
	}

	sort.Slice(hours, func(i, j int) bool {
		return hours[i].StartOfHour.Before(hours[j].StartOfHour)
	})

	return hours, nil
}

type QuantitativeValueType string

const (
	TEMPERATURE                QuantitativeValueType = "temperature"
	HUMIDITY                   QuantitativeValueType = "relativeHumidity"
	QUANTITATIVE_PRECIPITATION QuantitativeValueType = "quantitativePrecipitation"
	PRECIPITATION_PROBABILITY  QuantitativeValueType = "probabilityOfPrecipitation"
	SKY_COVER                  QuantitativeValueType = "skyCover"
	SNOWFALL_AMOUNT            QuantitativeValueType = "snowfallAmount"
	WIND_DIRECTION             QuantitativeValueType = "windDirection"
	WIND_SPEED                 QuantitativeValueType = "windSpeed"
)

func processQuantitativeValues(valueType QuantitativeValueType, additionalProperties map[string]noaaTypes.GridpointQuantitativeValueLayer, hoursMap map[string]*HourlyForecast) error {
	var value noaaTypes.GridpointQuantitativeValueLayer

	switch valueType {
	case TEMPERATURE:
		value = additionalProperties[string(TEMPERATURE)]
	case HUMIDITY:
		value = additionalProperties[string(HUMIDITY)]
	case QUANTITATIVE_PRECIPITATION:
		value = additionalProperties[string(QUANTITATIVE_PRECIPITATION)]
	case PRECIPITATION_PROBABILITY:
		value = additionalProperties[string(PRECIPITATION_PROBABILITY)]
	case SKY_COVER:
		value = additionalProperties[string(SKY_COVER)]
	case SNOWFALL_AMOUNT:
		value = additionalProperties[string(SNOWFALL_AMOUNT)]
	case WIND_DIRECTION:
		value = additionalProperties[string(WIND_DIRECTION)]
	case WIND_SPEED:
		value = additionalProperties[string(WIND_SPEED)]
	default:
		return errors.New("invalid value type")
	}

	for _, valueIter := range value.Values {
		time, err := validTimeToTime(valueIter.ValidTime)
		if err != nil {
			return err
		}
		stringifiedTime := time.String()
		hourObject := hoursMap[stringifiedTime]
		if hourObject == nil {
			hoursMap[stringifiedTime] = &HourlyForecast{
				StartOfHour: *time,
			}
			hourObject = hoursMap[stringifiedTime]
		}

		switch valueType {
		case TEMPERATURE:
			hourObject.TemperatureFahrenheit = celsiusToFahrenheit(valueIter.Value)
		case HUMIDITY:
			hourObject.Humidity = valueIter.Value
		case QUANTITATIVE_PRECIPITATION:
			hourObject.TotalPrecipMm = valueIter.Value
		case PRECIPITATION_PROBABILITY:
			hourObject.ChanceOfPrecip = valueIter.Value
		case SKY_COVER:
			hourObject.skyCoverPercent = valueIter.Value
		case SNOWFALL_AMOUNT:
			hourObject.snowfallMm = valueIter.Value
		case WIND_DIRECTION:
			hourObject.windDegrees = valueIter.Value
		case WIND_SPEED:
			hourObject.windKmH = valueIter.Value
		default:
			return errors.New("invalid value type")
		}
	}

	return nil
}

func validTimeToTime(validTime noaaTypes.ISO8601Interval) (*time.Time, error) {
	timeString, err := validTime.AsISO8601Interval0()
	if err != nil {
		return nil, err
	}

	amountToTrim := indexAt(timeString, "/", 0) //len(timeString) - 5
	// ValidTimes look like 2023-04-27T07:00:00+00:00/PT5H
	// the last 5 characters indicate the duration that the ValidTime represents(in this example, 5 hours)
	// in order to convert these to time objects we need to truncate
	timeStringTruncated := string([]byte(timeString)[:amountToTrim])
	time, err := time.Parse(time.RFC3339, timeStringTruncated)
	if err != nil {
		return nil, err
	}

	return &time, nil
}

func indexAt(s, sep string, n int) int {
	idx := strings.Index(s[n:], sep)
	if idx > -1 {
		idx += n
	}
	return idx
}

func celsiusToFahrenheit(celsius *float32) *float32 {
	fahrenheit := (*celsius * 9 / 5) + 32
	return &fahrenheit
}

func getPointerValueWithDefault(value *float32) float32 {
	if value == nil {
		return 0
	}

	return *value
}
