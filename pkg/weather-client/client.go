package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	noaaTypes "github.com/dsauerbrun/cragcast/pkg/weather-client/noaa"
)

const (
	// used for retrieval in the noaa-client
	weatherDotGovAPIEndpoint = "https://api.weather.gov"
)

type WeatherClient interface {
	GetForecast(lat, lng float64) (*noaaTypes.GridpointGeoJson, error)
}

type NoaaClient struct {
	APIEndpoint string
}

// New returns an instance of WeatherClient, which can be used to access weather data.
func New() WeatherClient {
	// TODO(joshrosso): This abstraction atop NoaaClient generation sits
	// here to support future weather providers or even the ability to mix
	// calls from various providers should we need to.
	return newNoaaClient()
}

func (nc *NoaaClient) GetForecast(lat, lng float64) (*noaaTypes.GridpointGeoJson, error) {
	// 40.0294122,-105.3223779 is lat,lng for boulder
	pointData, err := nc.getNoaaPointInfo(lat, lng)
	if err != nil {
		return nil, err
	}

	// BOU weather office(boulder), 52X, 75Y is boulder
	response, err := http.Get(*pointData.Properties.ForecastGridData)
	if err != nil {
		return nil, err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	forecastData := &noaaTypes.GridpointGeoJson{}
	err = json.Unmarshal(responseBody, forecastData)
	if err != nil {
		return nil, err
	}
	return forecastData, nil
}

func newNoaaClient() *NoaaClient {
	return &NoaaClient{
		APIEndpoint: weatherDotGovAPIEndpoint,
	}
}

func (nc *NoaaClient) getNoaaPointInfo(lat, lng float64) (*noaaTypes.PointGeoJson, error) {
	// cache this in the future as it will basically never change
	pointUrl := fmt.Sprintf("%s/points/%f,%f", nc.APIEndpoint, lat, lng)
	response, err := http.Get(pointUrl)
	if err != nil {
		return nil, err
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	pointData := &noaaTypes.PointGeoJson{}
	err = json.Unmarshal(responseBody, pointData)
	if err != nil {
		return nil, err
	}

	return pointData, nil
}
