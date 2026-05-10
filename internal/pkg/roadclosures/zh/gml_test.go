package zh

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"
)

func wrap(inner string) []byte {
	return []byte(`<g xmlns:gml="http://www.opengis.net/gml/3.2">` + inner + `</g>`)
}

func TestDecodeGeometry_PointWGS84Swap(t *testing.T) {
	g, err := decodeGeometry(wrap(`<gml:Point srsName="urn:ogc:def:crs:EPSG::4326"><gml:pos>47.226451 8.677549</gml:pos></gml:Point>`))
	require.NoError(t, err)
	pt, ok := g.Geometry().(orb.Point)
	require.True(t, ok)
	require.InDelta(t, 8.677549, pt.Lon(), 1e-9)
	require.InDelta(t, 47.226451, pt.Lat(), 1e-9)
}

func TestDecodeGeometry_PointNoSwap(t *testing.T) {
	g, err := decodeGeometry(wrap(`<gml:Point srsName="urn:ogc:def:crs:EPSG::2056"><gml:pos>2693650.34 1231525.97</gml:pos></gml:Point>`))
	require.NoError(t, err)
	pt, ok := g.Geometry().(orb.Point)
	require.True(t, ok)
	require.InDelta(t, 2693650.34, pt.Lon(), 1e-6)
	require.InDelta(t, 1231525.97, pt.Lat(), 1e-6)
}

func TestDecodeGeometry_LineStringSwap(t *testing.T) {
	g, err := decodeGeometry(wrap(`<gml:LineString srsName="urn:ogc:def:crs:EPSG::4326"><gml:posList>47.0 8.0 47.1 8.1 47.2 8.2</gml:posList></gml:LineString>`))
	require.NoError(t, err)
	ls, ok := g.Geometry().(orb.LineString)
	require.True(t, ok)
	require.Len(t, ls, 3)
	require.InDelta(t, 8.0, ls[0].Lon(), 1e-9)
	require.InDelta(t, 47.0, ls[0].Lat(), 1e-9)
	require.InDelta(t, 8.2, ls[2].Lon(), 1e-9)
	require.InDelta(t, 47.2, ls[2].Lat(), 1e-9)
}

func TestDecodeGeometry_PolygonSwap(t *testing.T) {
	g, err := decodeGeometry(wrap(`<gml:Polygon srsName="urn:ogc:def:crs:EPSG::4326">
		<gml:exterior>
			<gml:LinearRing>
				<gml:posList>47.226451 8.677549 47.226589 8.677362 47.226746 8.677126 47.226451 8.677549</gml:posList>
			</gml:LinearRing>
		</gml:exterior>
	</gml:Polygon>`))
	require.NoError(t, err)
	poly, ok := g.Geometry().(orb.Polygon)
	require.True(t, ok)
	require.Len(t, poly, 1)
	require.Len(t, poly[0], 4)
	require.InDelta(t, 8.677549, poly[0][0].Lon(), 1e-9)
	require.InDelta(t, 47.226451, poly[0][0].Lat(), 1e-9)
}

func TestDecodeGeometry_PolygonWithInterior(t *testing.T) {
	g, err := decodeGeometry(wrap(`<gml:Polygon srsName="urn:ogc:def:crs:EPSG::4326">
		<gml:exterior>
			<gml:LinearRing>
				<gml:posList>0 0 0 10 10 10 10 0 0 0</gml:posList>
			</gml:LinearRing>
		</gml:exterior>
		<gml:interior>
			<gml:LinearRing>
				<gml:posList>3 3 3 7 7 7 7 3 3 3</gml:posList>
			</gml:LinearRing>
		</gml:interior>
	</gml:Polygon>`))
	require.NoError(t, err)
	poly, ok := g.Geometry().(orb.Polygon)
	require.True(t, ok)
	require.Len(t, poly, 2)
}

func TestDecodeGeometry_Empty(t *testing.T) {
	g, err := decodeGeometry(nil)
	require.NoError(t, err)
	require.Nil(t, g)
}

func TestDecodeGeometry_OddPosList(t *testing.T) {
	_, err := decodeGeometry(wrap(`<gml:LineString srsName="urn:ogc:def:crs:EPSG::4326"><gml:posList>47.0 8.0 47.1</gml:posList></gml:LineString>`))
	require.Error(t, err)
}

func TestDecodeGeometry_Unsupported(t *testing.T) {
	_, err := decodeGeometry(wrap(`<gml:Curve srsName="urn:ogc:def:crs:EPSG::4326"/>`))
	require.Error(t, err)
}
