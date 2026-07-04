package api

import (
	"net/http"

	"jo-m.ch/go/cartomancer/internal/pkg/app"
)

type appConfigResponse struct {
	ExternalBaseURL     string `json:"externalBaseUrl"`
	InstanceName        string `json:"instanceName"`
	RegistrationEnabled bool   `json:"registrationEnabled"`
	DemoMode            bool   `json:"demoMode"`
	DemoEmail           string `json:"demoEmail,omitempty"`
	DemoPassword        string `json:"demoPassword,omitempty"`
	ContactEmail        string `json:"contactEmail"`
}

// handleGetAppConfig returns public application configuration.
func (sv *server) handleGetAppConfig(w http.ResponseWriter, _ *http.Request) {
	resp := appConfigResponse{
		ExternalBaseURL:     sv.appConfig.ExternalBaseURL,
		InstanceName:        sv.appConfig.InstanceName,
		RegistrationEnabled: sv.appConfig.RegistrationEnabled,
		DemoMode:            sv.appConfig.DemoMode,
		ContactEmail:        sv.appConfig.ContactEmail,
	}
	if sv.appConfig.DemoMode {
		resp.DemoEmail = app.DevInitialAdminEmail
		resp.DemoPassword = app.DevInitialAdminPassword
	}
	writeJSON(w, http.StatusOK, resp)
}
