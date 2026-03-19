package rest

import "net/http"

type appConfigResponse struct {
	ExternalBaseURL     string `json:"externalBaseUrl"`
	InstanceName        string `json:"instanceName"`
	TrackColor          string `json:"trackColor"`
	RegistrationEnabled bool   `json:"registrationEnabled"`
}

func (sv *server) handleGetAppConfig(w http.ResponseWriter, r *http.Request) {
	// TODO: Update OpenAPI YAML.
	writeJSON(w, http.StatusOK, appConfigResponse{
		ExternalBaseURL:     sv.appConfig.ExternalBaseURL,
		InstanceName:        sv.appConfig.InstanceName,
		TrackColor:          sv.appConfig.TrackColor,
		RegistrationEnabled: sv.appConfig.RegistrationEnabled,
	})
}
