package api

import (
	"net/http"
	"runtime/debug"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/geonames"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures"
)

type buildInfoModule struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

type versionResponse struct {
	GoVersion    string                  `json:"goVersion"`
	Path         string                  `json:"path"`
	Version      string                  `json:"version"`
	Deps         []buildInfoModule       `json:"deps"`
	Attributions []attribute.Attribution `json:"attributions"`
}

// handleGetVersion returns build information from the Go runtime and data source attributions.
func (sv *server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		writeError(w, http.StatusInternalServerError, "build info unavailable")
		return
	}

	resp := versionResponse{
		GoVersion: info.GoVersion,
		Path:      info.Main.Path,
		Version:   info.Main.Version,
		Deps:      make([]buildInfoModule, 0, len(info.Deps)),
		Attributions: []attribute.Attribution{
			meteo.DataAttribution,
			geonames.DataAttribution,
			roadclosures.DataAttribution,
		},
	}
	for _, dep := range info.Deps {
		resp.Deps = append(resp.Deps, buildInfoModule{
			Path:    dep.Path,
			Version: dep.Version,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}
