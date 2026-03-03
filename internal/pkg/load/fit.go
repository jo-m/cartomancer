package load

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"math"
	"path/filepath"
	"regexp"

	"github.com/muktihari/fit/decoder"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/typedef"
	"jo-m.ch/go/detour/internal/pkg/track"
)

func parseFitActivity(filename string, r io.ReadSeeker) (*filedef.Activity, error) {
	lis := filedef.NewListener()
	defer lis.Close()

	dec := decoder.New(r,
		decoder.WithMesgListener(lis),
		decoder.WithBroadcastOnly(),
	)

	if _, err := dec.CheckIntegrity(); err != nil {
		return nil, fmt.Errorf("integrity check failed on '%s': %w", filename, err)
	}

	_, err := r.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seek failed: %w", err)
	}

	var file *filedef.Activity = nil
	i := 0
	for dec.Next() {
		if i > 0 {
			panic("not handled")
		}

		_, err := dec.Decode()
		if err != nil {
			return nil, fmt.Errorf("failed to decode '%s': %w", filename, err)
		}

		var ok bool
		file, ok = lis.File().(*filedef.Activity)
		if !ok {
			return nil, fmt.Errorf("'%s' is not an activity file", filename)
		}
		i++
	}

	if file == nil {
		return nil, fmt.Errorf("failed to find FIT data in '%s'", filename)
	}

	return file, nil
}

type Activity struct {
	filename string
	act      *filedef.Activity
}

func (f *Activity) Filename() string {
	return f.filename
}

func loadFit(filename string, contents io.Reader) (track.TrackSource, error) {
	data, err := io.ReadAll(contents)
	if err != nil {
		return nil, fmt.Errorf("error reading FIT data: %v", err)
	}

	activity, err := parseFitActivity(filename, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing FIT file: %v", err)
	}

	return &Activity{
		filename: filename,
		act:      activity,
	}, nil
}

func semiCircleToDeg(s int32) float64 {
	return float64(s) / float64(0x80000000) * 180
}

func alt32(s uint32) float64 {
	return float64(s)/5 - 500
}

func (f *Activity) All() iter.Seq[track.Point] {
	return func(yield func(track.Point) bool) {
		for _, rec := range f.act.Records {
			if rec.PositionLat == math.MaxInt32 || rec.PositionLong == math.MaxInt32 {
				continue
			}

			point := track.Point{
				Time:      rec.Timestamp,
				Lat:       semiCircleToDeg(rec.PositionLat),
				Lon:       semiCircleToDeg(rec.PositionLong),
				Elevation: alt32(rec.EnhancedAltitude),
			}

			if !yield(point) {
				return
			}
		}
	}
}

var activityID = regexp.MustCompile(`[1-9][0-9]{5,}`)

func (f *Activity) Metadata() track.Metadata {
	ret := track.Metadata{
		Name:   filepath.Base(f.filename),
		Source: "Garmin",
		// Activities are always recorded.
		TrackType: track.TrackTypeRecorded,
	}

	if len(f.act.Sessions) != 1 {
		slog.Error("no session in activity", "filename", f.filename)
		return ret
	}

	sess := f.act.Sessions[0]

	switch sess.Sport {
	case typedef.SportRunning:
		ret.Sport = track.SportRunning
	case typedef.SportCycling:
		ret.Sport = track.SportCycling
	default:
		panic(fmt.Sprintf("unknown sport %d", sess.Sport))
	}

	if id := activityID.FindAllString(f.filename, -1); len(id) == 1 {
		ret.LinkURL = fmt.Sprintf("https://connect.garmin.com/activity/%s", id[0])
	}

	switch sess.SubSport {
	case typedef.SubSportTreadmill:
		ret.SubSport = track.SubSportRunningTreadmill
	case typedef.SubSportStreet:
		ret.SubSport = track.SubSportRunningOutdoor
	case typedef.SubSportTrail:
		ret.SubSport = track.SubSportRunningOutdoor
	case typedef.SubSportTrack:
		ret.SubSport = track.SubSportRunningOutdoor
	case typedef.SubSportSpin:
		ret.SubSport = track.SubSportCyclingSpinning
	case typedef.SubSportIndoorCycling:
		ret.SubSport = track.SubSportCyclingIndoorCycling
	case typedef.SubSportRoad:
		ret.SubSport = track.SubSportCyclingRoad
	case typedef.SubSportMountain:
		ret.SubSport = track.SubSportCyclingMountain
	case typedef.SubSportGravelCycling:
		ret.SubSport = track.SubSportCyclingGravel
	case typedef.SubSportCommuting:
		ret.SubSport = track.SubSportCyclingCommuting
	case typedef.SubSportGeneric:
		ret.SubSport = track.SubSportUnknown
	case typedef.SubSportInvalid:
		ret.SubSport = track.SubSportUnknown
	default:
		panic(fmt.Sprintf("unknown sub sport %d", sess.SubSport))
	}

	if sess.TotalAscent != math.MaxUint16 {
		ret.TotalAscentM = float64(sess.TotalAscent)
	}
	if sess.TotalDistance != math.MaxUint32 {
		ret.TotalDistanceM = float64(sess.TotalDistance) / 100
	}

	return ret
}
