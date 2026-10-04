// Package wfs is a minimal client for OGC Web Feature Service (WFS)
// endpoints. It supports the GetCapabilities and GetFeature operations and is
// intended for use cases where the full feature list of a single layer is
// downloaded in one go.
//
// Two GetFeature flavours are available:
//
//   - [Client.GetFeature] speaks WFS 2.0 for GML responses. Each feature is
//     kept as raw XML in [Feature.InnerXML]; callers decode the
//     schema-specific payload themselves with encoding/xml.
//   - [Client.GetFeatureGeoJSON] speaks WFS 2.0 or 1.1 for GeoJSON responses.
//     Each feature's properties are kept as raw JSON; callers decode them
//     with encoding/json.
//
// Usage:
//
//	ctx := context.Background()
//	client := wfs.NewClient("https://maps.zh.ch/wfs/TbaBaustellenZHWFS")
//	caps, err := client.GetCapabilities(ctx)
//	members, err := client.GetFeature(ctx, wfs.GetFeatureParams{
//	    TypeNames: "ms:baustellen-uebersicht",
//	    Count:     500,
//	})
package wfs

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"jo-m.ch/go/cartomancer/internal/pkg/client"
)

// WFS protocol versions supported by [Client].
const (
	// Version2 is WFS 2.0.0, spoken by clients from [NewClient].
	Version2 = "2.0.0"
	// Version11 is WFS 1.1.0, spoken by clients from [NewClientV11].
	// Only the GeoJSON GetFeature path supports it.
	Version11 = "1.1.0"
)

// userAgent identifies this client to upstream servers. Some deployments
// (e.g. the City of Zurich's WFS) reject Go's default "Go-http-client" user
// agent with a 403.
const userAgent = "cartomancer (+https://github.com/jo-m/cartomancer)"

// Client is a WFS client bound to a single service endpoint.
type Client struct {
	baseURL string
	version string
}

// NewClient creates a new Client for the given service endpoint URL, speaking
// WFS [Version2]. The URL should be the bare service path without query
// parameters (e.g. "https://maps.zh.ch/wfs/TbaBaustellenZHWFS"); any trailing
// '?' or '/' is trimmed.
func NewClient(baseURL string) Client {
	return newClientVersion(baseURL, Version2)
}

// NewClientV11 creates a new Client speaking WFS [Version11]. Only the
// GeoJSON GetFeature path ([Client.GetFeatureGeoJSON]) supports that version.
func NewClientV11(baseURL string) Client {
	return newClientVersion(baseURL, Version11)
}

// newClientVersion creates a Client speaking the given WFS version.
func newClientVersion(baseURL, version string) Client {
	return Client{baseURL: strings.TrimRight(baseURL, "/?"), version: version}
}

// requestURL builds a service URL with the given query parameters merged in
// alongside the standard service=WFS and version pair.
func (c *Client) requestURL(q url.Values) string {
	q.Set("service", "WFS")
	q.Set("version", c.version)
	return c.baseURL + "?" + q.Encode()
}

// fetchXML executes a GET request, expects an XML response, and decodes it
// into out. If the server returns an OWS ExceptionReport, fetchXML returns
// an [*ExceptionReport] as the error.
func fetchXML(ctx context.Context, reqURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/xml,text/xml")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.New().Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if exc := parseException(body); exc != nil {
		return exc
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d for %s", resp.StatusCode, reqURL)
	}

	if err := xml.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// parseException tries to decode body as an OWS ExceptionReport. Returns nil
// when the body is not an exception report.
func parseException(body []byte) *ExceptionReport {
	// Cheap pre-check: the root element of an exception report contains
	// "ExceptionReport". Avoids running the XML decoder over every body.
	if !bytes.Contains(body, []byte("ExceptionReport")) {
		return nil
	}
	var rep ExceptionReport
	if err := xml.Unmarshal(body, &rep); err != nil {
		return nil
	}
	if len(rep.Exceptions) == 0 {
		return nil
	}
	return &rep
}

// GetCapabilities fetches and parses the service's capabilities document.
// Only WFS 2.0 servers are supported.
func (c *Client) GetCapabilities(ctx context.Context) (*Capabilities, error) {
	if c.version != Version2 {
		return nil, fmt.Errorf("wfs: GetCapabilities requires WFS %s, client speaks %s", Version2, c.version)
	}
	u := c.requestURL(url.Values{"request": []string{"GetCapabilities"}})
	var caps Capabilities
	if err := fetchXML(ctx, u, &caps); err != nil {
		return nil, fmt.Errorf("get capabilities: %w", err)
	}
	return &caps, nil
}

// GetFeatureParams holds parameters for [Client.GetFeature] and
// [Client.GetFeatureGeoJSON].
type GetFeatureParams struct {
	// TypeNames identifies the layer to query, in the form advertised by
	// GetCapabilities (e.g. "ms:baustellen-uebersicht"). Required. It is
	// rendered as the "typeNames" (WFS 2.0) or "typeName" (WFS 1.1) query
	// parameter, depending on the client's version.
	TypeNames string
	// Count is the page size used while paginating through results.
	// When zero, the server's default is used; [Client.GetFeatureGeoJSON]
	// requires a positive value.
	Count int
	// SRSName overrides the response CRS. When empty, the layer's
	// DefaultCRS is used. Must be given in the form expected by the
	// service's WFS version (for example the URN form
	// "urn:ogc:def:crs:EPSG::4326" for WFS 2.0, or the short form
	// "EPSG:4326" for WFS 1.1).
	SRSName string
}

// GetFeature fetches every feature of a layer as GML, transparently following
// the paginated 'next' link until the server returns no more features. The
// returned slice preserves server order. Only WFS 2.0 servers are supported;
// use [Client.GetFeatureGeoJSON] for GeoJSON responses.
func (c *Client) GetFeature(ctx context.Context, params GetFeatureParams) ([]Member, error) {
	if c.version != Version2 {
		return nil, fmt.Errorf("wfs: GetFeature requires WFS %s, client speaks %s", Version2, c.version)
	}
	if params.TypeNames == "" {
		return nil, errors.New("wfs: GetFeature requires TypeNames")
	}

	q := url.Values{}
	q.Set("request", "GetFeature")
	q.Set("typeNames", params.TypeNames)
	if params.Count > 0 {
		q.Set("count", strconv.Itoa(params.Count))
	}
	if params.SRSName != "" {
		q.Set("srsName", params.SRSName)
	}
	pageURL := c.requestURL(q)

	var all []Member
	for pageURL != "" {
		var fc FeatureCollection
		if err := fetchXML(ctx, pageURL, &fc); err != nil {
			return nil, fmt.Errorf("get feature page: %w", err)
		}
		all = append(all, fc.Members...)
		// Stop if the server reports no progress to avoid infinite loops
		// when 'next' points to an effectively-empty page.
		if fc.NumberReturned == 0 {
			break
		}
		pageURL = fc.Next
	}
	return all, nil
}
