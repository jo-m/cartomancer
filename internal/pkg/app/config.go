// Package app contains configuration relating to the entire application.
package app

// AppConfig contains application-wide configuration.
type AppConfig struct {
	// AppName is the name of the application.
	// Used e.g. as issuer name for tokens.
	AppName string `arg:"--app-name,env:APP_NAME" default:"" help:"Name of the application, used as issuer in tokens"`
	// ExternalBaseURL is the URL at which the application is reachable from the outside.
	ExternalBaseURL string `arg:"--external-base-url,env:EXTERNAL_BASE_URL" default:"" help:"Base URL of the application"`
	// DevelopmentMode enables some development features.
	DevelopmentMode bool `arg:"--dev-mode,env:DEV_MODE" default:"false" help:"Enable development mode"`
}
