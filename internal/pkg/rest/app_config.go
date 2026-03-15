package rest

import "net/http"

type appConfigResponse struct {
	ExternalBaseURL string `json:"externalBaseUrl"`
	AppName         string `json:"appName"`
	TrackColor      string `json:"trackColor"`
}

func (sv *server) handleGetAppConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, appConfigResponse{
		ExternalBaseURL: sv.appConfig.ExternalBaseURL,
		AppName:         sv.appConfig.AppName,
		TrackColor:      sv.appConfig.TrackColor,
	})
}
