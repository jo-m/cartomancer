package api

import (
	"net/http"

	"jo-m.ch/go/cartomancer/internal/pkg/app"
)

type appConfigResponse struct {
	ExternalBaseURL     string `json:"externalBaseUrl"`
	InstanceName        string `json:"instanceName"`
	TrackColor          string `json:"trackColor"`
	RegistrationEnabled bool   `json:"registrationEnabled"`
	DemoMode            bool   `json:"demoMode"`
	DemoEmail           string `json:"demoEmail,omitempty"`
	DemoPassword        string `json:"demoPassword,omitempty"`
}

// handleGetAppConfig returns public application configuration.
func (sv *server) handleGetAppConfig(w http.ResponseWriter, r *http.Request) {
	resp := appConfigResponse{
		ExternalBaseURL:     sv.appConfig.ExternalBaseURL,
		InstanceName:        sv.appConfig.InstanceName,
		TrackColor:          sv.appConfig.TrackColor,
		RegistrationEnabled: sv.appConfig.RegistrationEnabled,
		DemoMode:            sv.appConfig.DemoMode,
	}
	if sv.appConfig.DemoMode {
		resp.DemoEmail = app.DevInitialAdminEmail
		resp.DemoPassword = app.DevInitialAdminPassword
	}
	writeJSON(w, http.StatusOK, resp)
}
