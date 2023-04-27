package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	cragcast "github.com/dsauerbrun/cragcast/internal/services/cragcast"
)

const (
	errorResponseTemplate = `{ "error": "%s" }`
)

type Controllers struct {
}

func (c *Controllers) GetForecast(w http.ResponseWriter, r *http.Request) {
	forecast, err := cragcast.GetForecast(1)
	if err != nil {
		// TODO(joshrosso): need to be more thoughtful with the error
		// message we pass back in the body
		respondWithInternalServerError(w, err)
		return
	}

	forecastJSON, err := json.Marshal(*forecast)
	if err != nil {
		// TODO(joshrosso): need to be more thoughtful with the error
		// message we pass back in the body
		respondWithInternalServerError(w, err)
		return
	}
	_, err = w.Write(forecastJSON)
	if err != nil {
		// TODO(joshrosso): need to be more thoughtful with the error
		// message we pass back in the body
		respondWithInternalServerError(w, err)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(forecastJSON))
}

func NewController() Controllers {
	return Controllers{}
}

func respondWithInternalServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(fmt.Sprintf(errorResponseTemplate, err.Error())))
}
