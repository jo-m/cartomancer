package api

import (
	"net/http"
	"runtime/debug"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/geonames"
	"jo-m.ch/go/cartomancer/internal/pkg/maps"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures/astra"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures/sz"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures/zh"
)

// buildVersion overrides the version reported by the /version endpoint.
// Set at build time via -ldflags "-X jo-m.ch/go/cartomancer/internal/pkg/api.buildVersion=v1.2.3".
// When empty, falls back to the module version from debug.ReadBuildInfo.
var buildVersion string

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
func (sv *server) handleGetVersion(w http.ResponseWriter, _ *http.Request) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		writeError(w, http.StatusInternalServerError, "build info unavailable")
		return
	}

	version := buildVersion
	if version == "" {
		version = info.Main.Version
	}

	resp := versionResponse{
		GoVersion: info.GoVersion,
		Path:      info.Main.Path,
		Version:   version,
		Deps:      make([]buildInfoModule, 0, len(info.Deps)),
		Attributions: []attribute.Attribution{
			meteo.DataAttribution,
			geonames.DataAttribution,
			astra.DataAttribution,
			zh.DataAttribution,
			sz.DataAttribution,
			maps.DataAttribution,
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
