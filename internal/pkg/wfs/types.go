package wfs

import "encoding/xml"

// Capabilities is the parsed WFS 2.0 GetCapabilities response.
// Only the subset useful for discovering layers (FeatureTypes) and basic
// service metadata is decoded.
type Capabilities struct {
	XMLName               xml.Name              `xml:"http://www.opengis.net/wfs/2.0 WFS_Capabilities"`
	Version               string                `xml:"version,attr"`
	ServiceIdentification ServiceIdentification `xml:"http://www.opengis.net/ows/1.1 ServiceIdentification"`
	FeatureTypes          []FeatureType         `xml:"http://www.opengis.net/wfs/2.0 FeatureTypeList>FeatureType"`
}

// ServiceIdentification holds the basic service-level metadata from a
// GetCapabilities response.
type ServiceIdentification struct {
	Title              string `xml:"http://www.opengis.net/ows/1.1 Title"`
	Abstract           string `xml:"http://www.opengis.net/ows/1.1 Abstract"`
	ServiceType        string `xml:"http://www.opengis.net/ows/1.1 ServiceType"`
	ServiceTypeVersion string `xml:"http://www.opengis.net/ows/1.1 ServiceTypeVersion"`
	Fees               string `xml:"http://www.opengis.net/ows/1.1 Fees"`
	AccessConstraints  string `xml:"http://www.opengis.net/ows/1.1 AccessConstraints"`
}

// FeatureType describes one layer advertised by the server.
type FeatureType struct {
	// Name is the qualified type name (e.g. "ms:baustellen-uebersicht").
	// Use it as TypeNames when calling GetFeature.
	Name string `xml:"http://www.opengis.net/wfs/2.0 Name"`
	// Title is a human-readable label.
	Title string `xml:"http://www.opengis.net/wfs/2.0 Title"`
	// Abstract is a free-text description. May be empty.
	Abstract string `xml:"http://www.opengis.net/wfs/2.0 Abstract"`
	// DefaultCRS is the CRS used by the server when no srsName is requested,
	// in URN form (e.g. "urn:ogc:def:crs:EPSG::2056").
	DefaultCRS string `xml:"http://www.opengis.net/wfs/2.0 DefaultCRS"`
	// OtherCRS lists CRSes the server can reproject the layer into.
	OtherCRS []string `xml:"http://www.opengis.net/wfs/2.0 OtherCRS"`
	// OutputFormats lists MIME types the server can return for this layer.
	OutputFormats []string `xml:"http://www.opengis.net/wfs/2.0 OutputFormats>Format"`
	// WGS84BoundingBox is the layer's extent in EPSG:4326.
	WGS84BoundingBox BBox `xml:"http://www.opengis.net/ows/1.1 WGS84BoundingBox"`
	// MetadataURL is an optional link to a metadata document.
	MetadataURL Link `xml:"http://www.opengis.net/wfs/2.0 MetadataURL"`
}

// BBox is an OWS-style bounding box with corners encoded as space-separated
// coordinate pairs (axis order depends on the CRS).
type BBox struct {
	LowerCorner string `xml:"http://www.opengis.net/ows/1.1 LowerCorner"`
	UpperCorner string `xml:"http://www.opengis.net/ows/1.1 UpperCorner"`
}

// Link carries an xlink:href attribute, used for service-provided URLs.
type Link struct {
	Href string `xml:"http://www.w3.org/1999/xlink href,attr"`
}

// FeatureCollection is the parsed envelope of a WFS 2.0 GetFeature response.
// Per-feature payloads are kept as raw XML in [Member.Feature] so callers can
// decode their own schema-specific types.
type FeatureCollection struct {
	XMLName   xml.Name `xml:"http://www.opengis.net/wfs/2.0 FeatureCollection"`
	TimeStamp string   `xml:"timeStamp,attr"`
	// NumberMatched is the total number of features matching the query.
	// MapServer returns the literal "unknown" when it cannot determine the
	// total, so this is kept as a string.
	NumberMatched string `xml:"numberMatched,attr"`
	// NumberReturned is the number of features in this page.
	NumberReturned int `xml:"numberReturned,attr"`
	// Next is the URL to fetch the next page, or empty when there are no
	// more results.
	Next    string   `xml:"next,attr"`
	Members []Member `xml:"http://www.opengis.net/wfs/2.0 member"`
}

// Member wraps a single feature inside a FeatureCollection.
type Member struct {
	Feature Feature `xml:",any"`
}

// Feature is one feature returned by GetFeature. Its element name and gml:id
// are captured; the full inner XML is preserved so callers can parse
// schema-specific attributes themselves.
type Feature struct {
	// XMLName carries the feature's element namespace and local name
	// (for example {http://mapserver.gis.umn.edu/mapserver baustellen-uebersicht}).
	XMLName xml.Name
	// GMLID is the value of the feature's gml:id attribute.
	GMLID string `xml:"id,attr"`
	// InnerXML is the raw XML between the feature's opening and closing
	// tags, including geometry and property children.
	InnerXML []byte `xml:",innerxml"`
}

// ExceptionReport is the parsed body of an OWS exception response, returned
// by WFS servers when a request fails.
type ExceptionReport struct {
	XMLName    xml.Name    `xml:"http://www.opengis.net/ows/1.1 ExceptionReport"`
	Version    string      `xml:"version,attr"`
	Exceptions []Exception `xml:"http://www.opengis.net/ows/1.1 Exception"`
}

// Exception is one entry in an [ExceptionReport].
type Exception struct {
	Code    string   `xml:"exceptionCode,attr"`
	Locator string   `xml:"locator,attr"`
	Texts   []string `xml:"http://www.opengis.net/ows/1.1 ExceptionText"`
}

// Error implements the error interface, formatting an exception report as a
// human-readable message.
func (e *ExceptionReport) Error() string {
	if len(e.Exceptions) == 0 {
		return "wfs: empty exception report"
	}
	first := e.Exceptions[0]
	msg := "wfs: " + first.Code
	if first.Locator != "" {
		msg += " (" + first.Locator + ")"
	}
	if len(first.Texts) > 0 {
		msg += ": " + first.Texts[0]
	}
	return msg
}
