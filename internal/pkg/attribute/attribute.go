// Package attribute provides a standard TASL (Title, Author, Source, License)
// attribution struct for crediting CC-licensed data sources.
package attribute

// Attribution describes a CC-licensed data source using the TASL model
// (Title, Author, Source, License) plus a What field describing what the
// data is used for in this application.
type Attribution struct {
	// What describes what the attributed data is used for (e.g. "Weather Forecast Data (Switzerland)").
	What string `json:"what"`
	// Title is the name of the work. May be empty if no title is provided.
	Title string `json:"title"`
	// Author is the licensor or copyright holder of the work.
	Author string `json:"author"`
	// Source is a URL where the original work can be found.
	Source string `json:"source"`
	// License is the short identifier of the license (e.g. "CC BY 4.0").
	License string `json:"license"`
	// LicenseURL is a URL to the full license text.
	LicenseURL string `json:"licenseUrl"`
}
