package osm

import (
	"context"
	"encoding/gob"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"maps/backend/internal/utils"

	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
)

type Platform struct {
	Ref    string
	Level  string
	Mode   string
	IsEdge bool
	Lat    float64
	Lon    float64
	Pts    [][2]float32
}

type StopPosition struct {
	Ref         string
	Level       string
	Mode        string
	LineRef     string
	Headsigns   []string
	Lat         float64
	Lon         float64
	TanLat      float64
	TanLon      float64
	HasTangent  bool
	HasDirected bool
}

type TrackSeg struct {
	Lat1  float32
	Lon1  float32
	Lat2  float32
	Lon2  float32
	Level string
	Mode  string
}

type PlatformQuery struct {
	Lat          float64
	Lon          float64
	Ref          string
	AllowNearest bool
	FromLat      float64
	FromLon      float64
	HasFrom      bool
	NextLat      float64
	NextLon      float64
	HasNext      bool
	Headsign     string
	Line         string
	Mode         string
}

type PlatformResult struct {
	Platform Platform
	TrackLat float64
	TrackLon float64
	HasTrack bool
	ExitSide string
	Found    bool
}

type gridKey struct {
	LatCell int
	LonCell int
}

type PlatformData struct {
	Platforms     []Platform
	StopPositions []StopPosition
	TrackSegs     []TrackSeg
}

const (
	defaultPBFPath   = "data/osm/ile-de-france-latest.osm.pbf"
	defaultCachePath = "data/processed/osm_platforms_v5.gob"
)

var (
	loadOnce     sync.Once
	grid         map[gridKey][]Platform
	stopPosGrid  map[gridKey][]StopPosition
	trackSegGrid map[gridKey][]TrackSeg
)

func EnsureLoaded() {
	loadOnce.Do(func() {
		if loaded, err := loadFromCache(defaultCachePath); err == nil && len(loaded.Platforms) > 0 {
			buildGrid(loaded)
			return
		}
		data, err := parsePBFPlatforms(defaultPBFPath)
		if err != nil {
			grid = make(map[gridKey][]Platform)
			stopPosGrid = make(map[gridKey][]StopPosition)
			trackSegGrid = make(map[gridKey][]TrackSeg)
			return
		}
		_ = saveToCache(defaultCachePath, data)
		buildGrid(data)
	})
}

func BuildGridForTest(data PlatformData) {
	loadOnce.Do(func() {})
	buildGrid(data)
}

func buildGrid(data PlatformData) {
	g := make(map[gridKey][]Platform, len(data.Platforms)/4)
	for _, p := range data.Platforms {
		k := gridKey{
			LatCell: int(math.Floor(p.Lat * 100)),
			LonCell: int(math.Floor(p.Lon * 100)),
		}
		g[k] = append(g[k], p)
	}
	grid = g

	sg := make(map[gridKey][]StopPosition, len(data.StopPositions)/4)
	for _, sp := range data.StopPositions {
		k := gridKey{
			LatCell: int(math.Floor(sp.Lat * 100)),
			LonCell: int(math.Floor(sp.Lon * 100)),
		}
		sg[k] = append(sg[k], sp)
	}
	stopPosGrid = sg

	tg := make(map[gridKey][]TrackSeg, len(data.TrackSegs)/4)
	for _, seg := range data.TrackSegs {
		midLat := (float64(seg.Lat1) + float64(seg.Lat2)) * 0.5
		midLon := (float64(seg.Lon1) + float64(seg.Lon2)) * 0.5
		k := gridKey{
			LatCell: int(math.Floor(midLat * 100)),
			LonCell: int(math.Floor(midLon * 100)),
		}
		tg[k] = append(tg[k], seg)
	}
	trackSegGrid = tg
}

func loadFromCache(path string) (PlatformData, error) {
	f, err := os.Open(path)
	if err != nil {
		return PlatformData{}, err
	}
	defer f.Close()
	var data PlatformData
	if err := gob.NewDecoder(f).Decode(&data); err != nil {
		return PlatformData{}, err
	}
	return data, nil
}

func saveToCache(path string, data PlatformData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(data)
}

func parsePBFPlatforms(path string) (PlatformData, error) {

	neededNodes := make(map[osm.NodeID]struct{}, 300_000)
	type rawStopNode struct {
		ref   string
		level string
		mode  string
		lat   float64
		lon   float64
	}
	type rawWay struct {
		id     osm.WayID
		ref    string
		level  string
		mode   string
		isEdge bool
		nodes  []osm.NodeID
	}
	type rawTrackWay struct {
		mode  string
		level string
		nodes []osm.NodeID
	}
	type rawRel struct {
		ref    string
		level  string
		mode   string
		isEdge bool
		ways   []osm.WayID
	}
	type rawRoute struct {
		mode    string
		ref     string
		to      string
		stopIDs []osm.NodeID
	}
	stopNodes := make(map[osm.NodeID]rawStopNode, 20_000)
	var platformWays []rawWay
	var trackWays []rawTrackWay
	wayNodesByID := make(map[osm.WayID][]osm.NodeID)
	var platformRels []rawRel
	var rawRoutes []rawRoute
	var platforms []Platform

	f1, err := os.Open(path)
	if err != nil {
		return PlatformData{}, err
	}
	scanner1 := osmpbf.New(context.Background(), f1, 4)
	for scanner1.Scan() {
		switch o := scanner1.Object().(type) {
		case *osm.Node:
			if isStopPositionTags(o.Tags) {
				stopNodes[o.ID] = rawStopNode{
					ref:   extractStopPositionRef(o.Tags),
					level: strings.TrimSpace(o.Tags.Find("level")),
					mode:  extractMode(o.Tags),
					lat:   o.Lat,
					lon:   o.Lon,
				}
			}
			if isPlatformTags(o.Tags) {
				platforms = append(platforms, Platform{
					Ref:    extractPlatformRef(o.Tags),
					Level:  strings.TrimSpace(o.Tags.Find("level")),
					Mode:   extractMode(o.Tags),
					IsEdge: o.Tags.Find("railway") == "platform_edge",
					Lat:    o.Lat,
					Lon:    o.Lon,
				})
			}
		case *osm.Way:
			if len(o.Nodes) == 0 {
				continue
			}
			isPlat := isPlatformTags(o.Tags)
			isTrack := len(o.Nodes) >= 2 && isRailTrackTags(o.Tags)
			if !isPlat && !isTrack {
				continue
			}
			ids := make([]osm.NodeID, len(o.Nodes))
			for i, wn := range o.Nodes {
				ids[i] = wn.ID
				neededNodes[wn.ID] = struct{}{}
			}
			if isPlat {
				platformWays = append(platformWays, rawWay{
					id:     o.ID,
					ref:    extractPlatformRef(o.Tags),
					level:  strings.TrimSpace(o.Tags.Find("level")),
					mode:   extractMode(o.Tags),
					isEdge: o.Tags.Find("railway") == "platform_edge",
					nodes:  ids,
				})
			}
			if isTrack {
				trackWays = append(trackWays, rawTrackWay{
					mode:  normalizeModeHint(o.Tags.Find("railway")),
					level: strings.TrimSpace(o.Tags.Find("level")),
					nodes: ids,
				})
			}
		case *osm.Relation:
			if isPlatformTags(o.Tags) {
				var memberWays []osm.WayID
				for _, m := range o.Members {
					if m.Type == osm.TypeWay {
						wID := osm.WayID(m.Ref)
						memberWays = append(memberWays, wID)
						wayNodesByID[wID] = nil
					}
				}
				if len(memberWays) > 0 {
					platformRels = append(platformRels, rawRel{
						ref:    extractPlatformRef(o.Tags),
						level:  strings.TrimSpace(o.Tags.Find("level")),
						mode:   extractMode(o.Tags),
						isEdge: o.Tags.Find("railway") == "platform_edge",
						ways:   memberWays,
					})
				}
			}
			if rMode := normalizeRouteMode(o.Tags.Find("route")); rMode != "" && o.Tags.Find("type") == "route" {
				var stops []osm.NodeID
				for _, m := range o.Members {
					if m.Type == osm.TypeNode && (m.Role == "stop" || strings.HasPrefix(m.Role, "stop_")) {
						stops = append(stops, osm.NodeID(m.Ref))
					}
				}
				if len(stops) >= 2 {
					rawRoutes = append(rawRoutes, rawRoute{
						mode:    rMode,
						ref:     strings.TrimSpace(o.Tags.Find("ref")),
						to:      strings.TrimSpace(o.Tags.Find("to")),
						stopIDs: stops,
					})
				}
			}
		}
	}
	err1 := scanner1.Err()
	scanner1.Close()
	f1.Close()
	if err1 != nil {
		return PlatformData{}, err1
	}

	if len(wayNodesByID) > 0 {
		fW, err := os.Open(path)
		if err != nil {
			return PlatformData{}, err
		}
		scannerW := osmpbf.New(context.Background(), fW, 4)
		for scannerW.Scan() {
			if w, ok := scannerW.Object().(*osm.Way); ok {
				if _, needed := wayNodesByID[w.ID]; needed && len(w.Nodes) > 0 {
					ids := make([]osm.NodeID, len(w.Nodes))
					for i, wn := range w.Nodes {
						ids[i] = wn.ID
						neededNodes[wn.ID] = struct{}{}
					}
					wayNodesByID[w.ID] = ids
				}
			}
		}
		errW := scannerW.Err()
		scannerW.Close()
		fW.Close()
		if errW != nil {
			return PlatformData{}, errW
		}
	}

	nodeCoords := make(map[osm.NodeID][2]float32, len(neededNodes))
	if len(neededNodes) > 0 {
		f2, err := os.Open(path)
		if err != nil {
			return PlatformData{}, err
		}
		scanner2 := osmpbf.New(context.Background(), f2, 4)
		for scanner2.Scan() {
			if n, ok := scanner2.Object().(*osm.Node); ok {
				if _, needed := neededNodes[n.ID]; needed {
					nodeCoords[n.ID] = [2]float32{float32(n.Lat), float32(n.Lon)}
				}
			}
		}
		err2 := scanner2.Err()
		scanner2.Close()
		f2.Close()
		if err2 != nil {
			return PlatformData{}, err2
		}
	}
	neededNodes = nil

	for _, pw := range platformWays {
		var sumLat, sumLon float64
		var pts [][2]float32
		for _, id := range pw.nodes {
			if c, ok := nodeCoords[id]; ok {
				sumLat += float64(c[0])
				sumLon += float64(c[1])
				pts = append(pts, c)
			}
		}
		if len(pts) > 0 {
			platforms = append(platforms, Platform{
				Ref:    pw.ref,
				Level:  pw.level,
				Mode:   pw.mode,
				IsEdge: pw.isEdge,
				Lat:    sumLat / float64(len(pts)),
				Lon:    sumLon / float64(len(pts)),
				Pts:    downsamplePlatformPts(pts, 64),
			})
		}
	}

	for _, pr := range platformRels {
		var sumLat, sumLon float64
		var count int
		var pts [][2]float32
		for _, wID := range pr.ways {
			var sub [][2]float32
			for _, nID := range wayNodesByID[wID] {
				if c, ok := nodeCoords[nID]; ok {
					sumLat += float64(c[0])
					sumLon += float64(c[1])
					count++
					sub = append(sub, c)
				}
			}
			if len(sub) > 0 {
				if len(pts) > 0 {
					pts = append(pts, [2]float32{float32(math.NaN()), float32(math.NaN())})
				}
				pts = append(pts, downsamplePlatformPts(sub, 32)...)
			}
		}
		if count > 0 {
			platforms = append(platforms, Platform{
				Ref:    pr.ref,
				Level:  pr.level,
				Mode:   pr.mode,
				IsEdge: pr.isEdge,
				Lat:    sumLat / float64(count),
				Lon:    sumLon / float64(count),
				Pts:    pts,
			})
		}
	}

	type nodeAxis struct {
		uLat  float64
		uLon  float64
		mode  string
		level string
	}
	nodeTrackAxes := make(map[osm.NodeID]nodeAxis, len(stopNodes))
	for _, tw := range trackWays {
		for idx, nid := range tw.nodes {
			sn, isSP := stopNodes[nid]
			if !isSP {
				continue
			}
			curr, okC := nodeCoords[nid]
			if !okC {
				continue
			}

			prevPt := curr
			for step := 1; step <= 6 && idx-step >= 0; step++ {
				if c, ok := nodeCoords[tw.nodes[idx-step]]; ok {
					prevPt = c
					if utils.CoordsGetDistance(float64(curr[1]), float64(curr[0]), float64(c[1]), float64(c[0]))*1000.0 >= 3.0 {
						break
					}
				}
			}
			nextPt := curr
			for step := 1; step <= 6 && idx+step < len(tw.nodes); step++ {
				if c, ok := nodeCoords[tw.nodes[idx+step]]; ok {
					nextPt = c
					if utils.CoordsGetDistance(float64(curr[1]), float64(curr[0]), float64(c[1]), float64(c[0]))*1000.0 >= 3.0 {
						break
					}
				}
			}
			dLat := float64(nextPt[0]) - float64(prevPt[0])
			dLon := float64(nextPt[1]) - float64(prevPt[1])
			uLat, uLon, okNorm := normalizeMetricDir(float64(curr[0]), dLat, dLon)
			if !okNorm {
				continue
			}
			prev, exists := nodeTrackAxes[nid]
			if !exists || (sn.mode != "" && prev.mode != sn.mode && tw.mode == sn.mode) {
				nodeTrackAxes[nid] = nodeAxis{uLat: uLat, uLon: uLon, mode: tw.mode, level: tw.level}
			}
		}
	}

	type spKey struct {
		nid  osm.NodeID
		sign int8
	}
	mergedMap := make(map[spKey]*StopPosition, len(stopNodes))
	directedNIDs := make(map[osm.NodeID]bool, len(stopNodes))

	for _, rt := range rawRoutes {
		for i, sid := range rt.stopIDs {
			sn, isSP := stopNodes[sid]
			if !isSP {
				continue
			}

			var rLat, rLon float64
			hasRouteVec := false
			if i > 0 {
				if prev, okP := stopNodes[rt.stopIDs[i-1]]; okP {
					rLat += sn.lat - prev.lat
					rLon += sn.lon - prev.lon
					hasRouteVec = true
				}
			}
			if i+1 < len(rt.stopIDs) {
				if next, okN := stopNodes[rt.stopIDs[i+1]]; okN {
					uL, uO, ok := normalizeMetricDir(sn.lat, next.lat-sn.lat, next.lon-sn.lon)
					if ok {
						if hasRouteVec {

							inL, inO, okIn := normalizeMetricDir(sn.lat, rLat, rLon)
							if okIn {
								rLat, rLon = inL+uL, inO+uO
							} else {
								rLat, rLon = uL, uO
							}
						} else {
							rLat, rLon = uL, uO
						}
						hasRouteVec = true
					}
				}
			}
			uRouteLat, uRouteLon, okR := normalizeMetricDir(sn.lat, rLat, rLon)
			if !hasRouteVec || !okR {
				continue
			}

			tanLat, tanLon := uRouteLat, uRouteLon
			sign := int8(1)
			axis, hasAxis := nodeTrackAxes[sid]
			if hasAxis {
				tanLat, tanLon = axis.uLat, axis.uLon
				cosLat := math.Cos(sn.lat * math.Pi / 180.0)
				dot := (axis.uLon*cosLat)*(uRouteLon*cosLat) + axis.uLat*uRouteLat
				if dot < 0 {
					tanLat = -tanLat
					tanLon = -tanLon
					sign = -1
				}
			} else {

				sign = quantizeHeadingSign(sn.lat, tanLat, tanLon)
			}

			key := spKey{nid: sid, sign: sign}
			sp, exists := mergedMap[key]
			if !exists {
				mode := sn.mode
				if mode == "" {
					mode = rt.mode
				}
				lvl := sn.level
				if lvl == "" && hasAxis {
					lvl = axis.level
				}
				sp = &StopPosition{
					Ref:         sn.ref,
					Level:       lvl,
					Mode:        mode,
					Lat:         sn.lat,
					Lon:         sn.lon,
					TanLat:      tanLat,
					TanLon:      tanLon,
					HasTangent:  true,
					HasDirected: true,
				}
				mergedMap[key] = sp
				directedNIDs[sid] = true
			}
			if rt.ref != "" && !tokenContains(sp.LineRef, rt.ref) {
				if sp.LineRef == "" {
					sp.LineRef = rt.ref
				} else {
					sp.LineRef += ";" + rt.ref
				}
			}
			if rt.to != "" {
				sp.Headsigns = appendUniqueStr(sp.Headsigns, rt.to)
			}
		}
	}

	var stopPositions []StopPosition
	for _, sp := range mergedMap {
		stopPositions = append(stopPositions, *sp)
	}
	for sid, sn := range stopNodes {
		if directedNIDs[sid] {
			continue
		}
		sp := StopPosition{
			Ref:   sn.ref,
			Level: sn.level,
			Mode:  sn.mode,
			Lat:   sn.lat,
			Lon:   sn.lon,
		}
		if axis, ok := nodeTrackAxes[sid]; ok {
			sp.TanLat = axis.uLat
			sp.TanLon = axis.uLon
			sp.HasTangent = true
			if sp.Mode == "" {
				sp.Mode = axis.mode
			}
			if sp.Level == "" {
				sp.Level = axis.level
			}
		}
		stopPositions = append(stopPositions, sp)
	}

	platBuckets := make(map[gridKey][][2]float64, len(platforms)/4)
	for _, p := range platforms {
		k := gridKey{LatCell: int(math.Floor(p.Lat * 100)), LonCell: int(math.Floor(p.Lon * 100))}
		platBuckets[k] = append(platBuckets[k], [2]float64{p.Lat, p.Lon})
	}
	var trackSegs []TrackSeg
	for _, tw := range trackWays {
		for k := 0; k+1 < len(tw.nodes); k++ {
			p1, ok1 := nodeCoords[tw.nodes[k]]
			p2, ok2 := nodeCoords[tw.nodes[k+1]]
			if !ok1 || !ok2 || (p1[0] == p2[0] && p1[1] == p2[1]) {
				continue
			}
			mLat := (float64(p1[0]) + float64(p2[0])) * 0.5
			mLon := (float64(p1[1]) + float64(p2[1])) * 0.5
			cLat := int(math.Floor(mLat * 100))
			cLon := int(math.Floor(mLon * 100))
			near := false
			for dl := -1; dl <= 1 && !near; dl++ {
				for do := -1; do <= 1 && !near; do++ {
					for _, pc := range platBuckets[gridKey{cLat + dl, cLon + do}] {
						if utils.CoordsGetDistance(mLon, mLat, pc[1], pc[0])*1000.0 <= 220.0 {
							near = true
							break
						}
					}
				}
			}
			if near {
				trackSegs = append(trackSegs, TrackSeg{
					Lat1:  p1[0],
					Lon1:  p1[1],
					Lat2:  p2[0],
					Lon2:  p2[1],
					Level: tw.level,
					Mode:  tw.mode,
				})
			}
		}
	}

	nodeCoords = nil
	platformWays = nil
	trackWays = nil
	rawRoutes = nil
	stopNodes = nil
	platBuckets = nil
	wayNodesByID = nil
	platformRels = nil
	runtime.GC()
	debug.FreeOSMemory()

	return PlatformData{
		Platforms:     platforms,
		StopPositions: stopPositions,
		TrackSegs:     trackSegs,
	}, nil
}

func downsamplePlatformPts(pts [][2]float32, maxPts int) [][2]float32 {
	if len(pts) <= maxPts || maxPts < 3 {
		return pts
	}
	out := make([][2]float32, 0, maxPts)
	step := float64(len(pts)-1) / float64(maxPts-1)
	for i := 0; i < maxPts-1; i++ {
		out = append(out, pts[int(math.Round(float64(i)*step))])
	}
	out = append(out, pts[len(pts)-1])
	return out
}

func quantizeHeadingSign(lat, tanLat, tanLon float64) int8 {
	cosLat := math.Cos(lat * math.Pi / 180.0)
	if math.Abs(tanLat) >= math.Abs(tanLon*cosLat) {
		if tanLat >= 0 {
			return 1
		}
		return -1
	}
	if tanLon >= 0 {
		return 2
	}
	return -2
}

func normalizeMetricDir(lat, dLat, dLon float64) (uLat, uLon float64, ok bool) {
	cosLat := math.Cos(lat * math.Pi / 180.0)
	lenM := math.Hypot(dLon*cosLat, dLat)
	if lenM < 1e-12 {
		return 0, 0, false
	}
	return dLat / lenM, dLon / lenM, true
}

func appendUniqueStr(slice []string, v string) []string {
	for _, existing := range slice {
		if existing == v {
			return slice
		}
	}
	return append(slice, v)
}

func tokenContains(list, target string) bool {
	nt := normalizePlatformRef(target)
	if nt == "" || list == "" {
		return false
	}
	for _, p := range strings.FieldsFunc(list, func(r rune) bool { return r == ';' || r == ',' || r == '/' }) {
		if normalizePlatformRef(p) == nt {
			return true
		}
	}
	return false
}

func isStopPositionTags(tags osm.Tags) bool {
	if tags.Find("public_transport") != "stop_position" {
		return false
	}
	if tags.Find("bus") == "yes" && tags.Find("train") != "yes" && tags.Find("subway") != "yes" && tags.Find("tram") != "yes" && tags.Find("light_rail") != "yes" {
		return false
	}
	return true
}

func isRailTrackTags(tags osm.Tags) bool {
	r := tags.Find("railway")
	if r != "subway" && r != "rail" && r != "tram" && r != "light_rail" {
		return false
	}
	svc := tags.Find("service")
	if svc == "yard" || svc == "crossover" || svc == "siding" {
		return false
	}
	return true
}

func isPlatformTags(tags osm.Tags) bool {
	r := tags.Find("railway")
	if r == "platform" || r == "platform_edge" {
		return true
	}
	if tags.Find("public_transport") == "platform" {
		if tags.Find("highway") == "bus_stop" {
			return false
		}
		if tags.Find("bus") == "yes" && tags.Find("tram") != "yes" && tags.Find("subway") != "yes" && tags.Find("train") != "yes" && tags.Find("light_rail") != "yes" {
			return false
		}
		return true
	}
	return false
}

func extractMode(tags osm.Tags) string {
	for _, key := range []string{"subway", "train", "tram", "light_rail"} {
		if tags.Find(key) == "yes" {
			return key
		}
	}
	if st := tags.Find("station"); st != "" {
		if m := normalizeModeHint(st); m != "" {
			return m
		}
	}
	if rw := tags.Find("railway"); rw != "" {
		if m := normalizeModeHint(rw); m != "" {
			return m
		}
	}
	return ""
}

func normalizeRouteMode(routeTag string) string {
	switch strings.ToLower(strings.TrimSpace(routeTag)) {
	case "subway":
		return "subway"
	case "train", "railway":
		return "train"
	case "tram":
		return "tram"
	case "light_rail":
		return "light_rail"
	}
	return ""
}

func normalizeModeHint(m string) string {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case "subway", "metro", "1":
		return "subway"
	case "train", "rail", "rer", "transilien", "national_rail", "regional_rail", "2":
		return "train"
	case "tram", "tramway", "0":
		return "tram"
	case "light_rail", "tram_train":
		return "light_rail"
	}
	return ""
}

func modeMatches(a, b string) bool {
	na, nb := normalizeModeHint(a), normalizeModeHint(b)
	if na == "" || nb == "" {
		return true
	}
	if na == nb {
		return true
	}

	if (na == "tram" || na == "light_rail") && (nb == "tram" || nb == "light_rail") {
		return true
	}
	return false
}

func extractStopPositionRef(tags osm.Tags) string {
	for _, key := range []string{"local_ref", "ref"} {
		if v := strings.TrimSpace(tags.Find(key)); v != "" {
			return v
		}
	}
	return ""
}

func extractPlatformRef(tags osm.Tags) string {
	for _, key := range []string{"local_ref", "ref", "name"} {
		if v := strings.TrimSpace(tags.Find(key)); v != "" {
			return v
		}
	}
	return ""
}

func FindPlatform(lat, lon float64, ref string) (Platform, bool) {
	return FindPlatformWithFallback(lat, lon, ref, false)
}

func FindPlatformWithFallback(lat, lon float64, ref string, allowNearest bool) (Platform, bool) {
	res := ResolvePlatform(PlatformQuery{
		Lat:          lat,
		Lon:          lon,
		Ref:          ref,
		AllowNearest: allowNearest,
	})
	return res.Platform, res.Found
}

func FindPlatformWithTrack(lat, lon float64, ref string, allowNearest bool) (plat Platform, trackLat, trackLon float64, hasTrack, found bool) {
	res := ResolvePlatform(PlatformQuery{
		Lat:          lat,
		Lon:          lon,
		Ref:          ref,
		AllowNearest: allowNearest,
	})
	return res.Platform, res.TrackLat, res.TrackLon, res.HasTrack, res.Found
}

func ResolvePlatform(q PlatformQuery) PlatformResult {
	EnsureLoaded()
	normRef := normalizePlatformRef(q.Ref)
	normMode := normalizeModeHint(q.Mode)
	normLine := normalizePlatformRef(q.Line)
	if len(grid) == 0 || (normRef == "" && !q.AllowNearest) {
		return PlatformResult{}
	}

	var travLat, travLon float64
	hasTrav := false
	if q.HasFrom {
		uL, uO, ok := normalizeMetricDir(q.Lat, q.Lat-q.FromLat, q.Lon-q.FromLon)
		if ok {
			travLat, travLon, hasTrav = uL, uO, true
		}
	} else if q.HasNext {
		uL, uO, ok := normalizeMetricDir(q.Lat, q.NextLat-q.Lat, q.NextLon-q.Lon)
		if ok {
			travLat, travLon, hasTrav = uL, uO, true
		}
	}

	if normRef != "" || hasTrav || q.Headsign != "" || normLine != "" {
		if sp, okSP := findBestStopPosition(q.Lat, q.Lon, normRef, normMode, normLine, q.Headsign, travLat, travLon, hasTrav, q.AllowNearest); okSP {
			if adjPlat, edgeLat, edgeLon, edgeDist, okAdj := findNearestPlatformNear(sp.Lat, sp.Lon, sp.Level, normRef, normMode, 150.0); okAdj {
				if sp.Ref != "" {
					adjPlat.Ref = sp.Ref
				}
				dirLat, dirLon := travLat, travLon
				hasDir := hasTrav
				if sp.HasTangent {
					dirLat, dirLon = sp.TanLat, sp.TanLon
					hasDir = true
					if hasTrav && (normRef != "" || !sp.HasDirected) {
						cosLat := math.Cos(sp.Lat * math.Pi / 180.0)
						if (dirLon*cosLat)*(travLon*cosLat)+dirLat*travLat < 0 {
							dirLat, dirLon = -dirLat, -dirLon
						}
					}
				}
				exitSide := ""
				if hasDir {
					tLat, tLon := edgeLat, edgeLon
					if edgeDist < 0.5 {
						tLat, tLon = adjPlat.Lat, adjPlat.Lon
					}
					exitSide = computeExitSideWithDir(sp.Lat, sp.Lon, dirLat, dirLon, tLat, tLon)
				}
				return PlatformResult{
					Platform: adjPlat,
					TrackLat: sp.Lat,
					TrackLon: sp.Lon,
					HasTrack: true,
					ExitSide: exitSide,
					Found:    true,
				}
			}
		}
	}

	plat, edgeLat, edgeLon, edgeDist, found := findBestPlatformDirect(q.Lat, q.Lon, normRef, normMode, q.AllowNearest)
	if !found {
		return PlatformResult{}
	}

	trackLat, trackLon := q.Lat, q.Lon
	hasTrack := false
	dirLat, dirLon := travLat, travLon
	hasDir := hasTrav

	tPlatLat, tPlatLon := edgeLat, edgeLon
	if edgeDist < 0.5 {
		tPlatLat, tPlatLon = plat.Lat, plat.Lon
	}
	if segPtLat, segPtLon, segDirLat, segDirLon, okSeg := findNearestTrackNear(tPlatLat, tPlatLon, normMode, plat.Level, travLat, travLon, hasTrav); okSeg {
		trackLat, trackLon = segPtLat, segPtLon
		hasTrack = true
		dirLat, dirLon = segDirLat, segDirLon
		hasDir = true

		if cLat, cLon, dEdge, okEdge := closestPointOnPlatform(plat, trackLat, trackLon); okEdge && dEdge >= 0.5 {
			tPlatLat, tPlatLon = cLat, cLon
		}
	}

	exitSide := ""
	if hasDir {
		exitSide = computeExitSideWithDir(trackLat, trackLon, dirLat, dirLon, tPlatLat, tPlatLon)
	}
	return PlatformResult{
		Platform: plat,
		TrackLat: trackLat,
		TrackLon: trackLon,
		HasTrack: hasTrack,
		ExitSide: exitSide,
		Found:    true,
	}
}

func findBestStopPosition(lat, lon float64, normRef, normMode, normLine, headsign string, travLat, travLon float64, hasTrav, allowNearest bool) (StopPosition, bool) {
	if normRef == "" && !allowNearest {
		return StopPosition{}, false
	}
	maxRadius := 600.0
	if normRef == "" {
		maxRadius = 180.0
	}

	centerLat := int(math.Floor(lat * 100))
	centerLon := int(math.Floor(lon * 100))
	cosLat := math.Cos(lat * math.Pi / 180.0)
	normHS := normalizeHeadsign(headsign)

	var best StopPosition
	bestScore := math.MaxFloat64
	found := false

	for dLat := -1; dLat <= 1; dLat++ {
		for dLon := -1; dLon <= 1; dLon++ {
			for _, sp := range stopPosGrid[gridKey{centerLat + dLat, centerLon + dLon}] {
				d := utils.CoordsGetDistance(lon, lat, sp.Lon, sp.Lat) * 1000.0
				if d > maxRadius {
					continue
				}
				if normRef != "" && !platformRefMatches(sp.Ref, normRef) {
					continue
				}

				score := d
				if normMode != "" && sp.Mode != "" {
					if modeMatches(sp.Mode, normMode) {
						score -= 120.0
					} else {
						score += 400.0
					}
				}
				if normLine != "" && sp.LineRef != "" {
					if tokenContains(sp.LineRef, normLine) {
						score -= 150.0
					} else {
						score += 250.0
					}
				}
				if normHS != "" && len(sp.Headsigns) > 0 {
					if headsignMatchesAny(normHS, sp.Headsigns) {
						score -= 260.0
					} else {
						score += 120.0
					}
				}
				if hasTrav && sp.HasTangent {
					dot := (sp.TanLon*cosLat)*(travLon*cosLat) + sp.TanLat*travLat
					if sp.HasDirected {
						if dot >= 0 {
							score -= 140.0 * dot
						} else {
							score += 320.0 * (-dot)
						}
					} else {

						score -= 40.0 * math.Abs(dot)
					}
				}

				if score < bestScore {
					bestScore = score
					best = sp
					found = true
				}
			}
		}
	}
	return best, found
}

func findBestPlatformDirect(lat, lon float64, normRef, normMode string, allowNearest bool) (best Platform, edgeLat, edgeLon, edgeDist float64, found bool) {
	centerLat := int(math.Floor(lat * 100))
	centerLon := int(math.Floor(lon * 100))

	bestRefScore := math.MaxFloat64
	var bestRef PlatCandidate
	bestRefFound := false

	bestNearScore := math.MaxFloat64
	var bestNear PlatCandidate
	bestNearFound := false

	for dLat := -1; dLat <= 1; dLat++ {
		for dLon := -1; dLon <= 1; dLon++ {
			for _, p := range grid[gridKey{centerLat + dLat, centerLon + dLon}] {
				eLat, eLon, dEdge, okEdge := closestPointOnPlatform(p, lat, lon)
				if !okEdge {
					eLat, eLon = p.Lat, p.Lon
					dEdge = utils.CoordsGetDistance(lon, lat, p.Lon, p.Lat) * 1000.0
				}
				modePen := 0.0
				if normMode != "" && p.Mode != "" {
					if modeMatches(p.Mode, normMode) {
						modePen = -80.0
					} else {
						modePen = 300.0
					}
				}
				edgePen := 0.0
				if p.IsEdge {
					edgePen = 20.0
				}
				if allowNearest && dEdge <= 120.0 {
					s := dEdge + modePen + edgePen
					if s < bestNearScore {
						bestNearScore = s
						bestNear = PlatCandidate{p: p, eLat: eLat, eLon: eLon, dEdge: dEdge}
						bestNearFound = true
					}
				}
				if normRef != "" && dEdge <= 600.0 && platformRefMatches(p.Ref, normRef) {
					s := dEdge + modePen + edgePen
					if s < bestRefScore {
						bestRefScore = s
						bestRef = PlatCandidate{p: p, eLat: eLat, eLon: eLon, dEdge: dEdge}
						bestRefFound = true
					}
				}
			}
		}
	}
	if bestRefFound {
		return bestRef.p, bestRef.eLat, bestRef.eLon, bestRef.dEdge, true
	}
	if allowNearest && bestNearFound {
		return bestNear.p, bestNear.eLat, bestNear.eLon, bestNear.dEdge, true
	}
	return Platform{}, 0, 0, 0, false
}

type PlatCandidate struct {
	p     Platform
	eLat  float64
	eLon  float64
	dEdge float64
}

func findNearestPlatformNear(lat, lon float64, prefLevel, normRef, normMode string, maxRadiusM float64) (best Platform, edgeLat, edgeLon, edgeDist float64, found bool) {
	centerLat := int(math.Floor(lat * 100))
	centerLon := int(math.Floor(lon * 100))
	bestScore := math.MaxFloat64
	for dLat := -1; dLat <= 1; dLat++ {
		for dLon := -1; dLon <= 1; dLon++ {
			for _, p := range grid[gridKey{centerLat + dLat, centerLon + dLon}] {
				eLat, eLon, dEdge, okEdge := closestPointOnPlatform(p, lat, lon)
				if !okEdge {
					eLat, eLon = p.Lat, p.Lon
					dEdge = utils.CoordsGetDistance(lon, lat, p.Lon, p.Lat) * 1000.0
				}
				if dEdge > maxRadiusM {
					continue
				}
				score := dEdge
				if p.IsEdge {
					score += 20.0
				}
				if prefLevel != "" && p.Level != "" {
					if p.Level == prefLevel {
						score -= 100.0
					} else {
						score += 100.0
					}
				}
				if normMode != "" && p.Mode != "" {
					if modeMatches(p.Mode, normMode) {
						score -= 80.0
					} else {
						score += 300.0
					}
				}
				if normRef != "" {
					if platformRefMatches(p.Ref, normRef) {
						score -= 500.0
					} else if p.Ref != "" {

						score += 250.0
					}
				}
				if score < bestScore {
					bestScore = score
					best = p
					edgeLat, edgeLon, edgeDist = eLat, eLon, dEdge
					found = true
				}
			}
		}
	}
	return best, edgeLat, edgeLon, edgeDist, found
}

func findNearestTrackNear(lat, lon float64, normMode, prefLevel string, travLat, travLon float64, hasTrav bool) (segLat, segLon, dirLat, dirLon float64, found bool) {
	if len(trackSegGrid) == 0 {
		return 0, 0, 0, 0, false
	}
	centerLat := int(math.Floor(lat * 100))
	centerLon := int(math.Floor(lon * 100))
	cosLat := math.Cos(lat * math.Pi / 180.0)
	bestScore := math.MaxFloat64

	for dLat := -1; dLat <= 1; dLat++ {
		for dLon := -1; dLon <= 1; dLon++ {
			for _, seg := range trackSegGrid[gridKey{centerLat + dLat, centerLon + dLon}] {
				cLat, cLon, dM := closestPointOnSegment(lat, lon, float64(seg.Lat1), float64(seg.Lon1), float64(seg.Lat2), float64(seg.Lon2))
				if dM > 60.0 {
					continue
				}
				uL, uO, okDir := normalizeMetricDir(lat, float64(seg.Lat2)-float64(seg.Lat1), float64(seg.Lon2)-float64(seg.Lon1))
				if !okDir {
					continue
				}
				score := dM
				if normMode != "" && seg.Mode != "" {
					if modeMatches(seg.Mode, normMode) {
						score -= 40.0
					} else {
						score += 200.0
					}
				}
				if prefLevel != "" && seg.Level != "" && seg.Level == prefLevel {
					score -= 30.0
				}
				if hasTrav {
					dot := math.Abs((uO*cosLat)*(travLon*cosLat) + uL*travLat)
					score -= 20.0 * dot
				}
				if score < bestScore {
					bestScore = score
					segLat, segLon = cLat, cLon
					dirLat, dirLon = uL, uO
					if hasTrav && (uO*cosLat)*(travLon*cosLat)+uL*travLat < 0 {
						dirLat, dirLon = -dirLat, -dirLon
					}
					found = true
				}
			}
		}
	}
	return segLat, segLon, dirLat, dirLon, found
}

func closestPointOnPlatform(p Platform, lat, lon float64) (bestLat, bestLon, bestDistM float64, ok bool) {
	if len(p.Pts) < 2 {
		return 0, 0, 0, false
	}
	bestDistM = math.MaxFloat64
	for i := 0; i+1 < len(p.Pts); i++ {
		a, b := p.Pts[i], p.Pts[i+1]
		if math.IsNaN(float64(a[0])) || math.IsNaN(float64(b[0])) {
			continue
		}
		cLat, cLon, dM := closestPointOnSegment(lat, lon, float64(a[0]), float64(a[1]), float64(b[0]), float64(b[1]))
		if dM < bestDistM {
			bestDistM = dM
			bestLat = cLat
			bestLon = cLon
			ok = true
		}
	}
	return bestLat, bestLon, bestDistM, ok
}

func closestPointOnSegment(pLat, pLon, aLat, aLon, bLat, bLon float64) (cLat, cLon, distM float64) {
	cosLat := math.Cos(pLat * math.Pi / 180.0)
	const mPerDeg = 111320.0
	ax := (aLon - pLon) * cosLat * mPerDeg
	ay := (aLat - pLat) * mPerDeg
	bx := (bLon - pLon) * cosLat * mPerDeg
	by := (bLat - pLat) * mPerDeg
	vx := bx - ax
	vy := by - ay
	len2 := vx*vx + vy*vy
	t := 0.0
	if len2 > 0 {
		t = -(ax*vx + ay*vy) / len2
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
	}
	cx := ax + t*vx
	cy := ay + t*vy
	distM = math.Hypot(cx, cy)
	cLat = aLat + t*(bLat-aLat)
	cLon = aLon + t*(bLon-aLon)
	return cLat, cLon, distM
}

func normalizeHeadsign(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	repl := strings.NewReplacer("–", " ", "—", " ", "-", " ", "'", " ", "’", " ", "é", "e", "è", "e", "ê", "e", "ë", "e", "à", "a", "â", "a", "î", "i", "ï", "i", "ô", "o", "ù", "u", "û", "u", "ç", "c")
	return strings.Join(strings.Fields(repl.Replace(s)), " ")
}

func headsignMatchesAny(normHS string, candidates []string) bool {
	if normHS == "" {
		return false
	}
	for _, c := range candidates {
		nc := normalizeHeadsign(c)
		if nc == "" {
			continue
		}
		if nc == normHS || strings.Contains(nc, normHS) || strings.Contains(normHS, nc) {
			return true
		}
	}
	return false
}

func platformRefMatches(osmRef, normTarget string) bool {
	if osmRef == "" || normTarget == "" {
		return false
	}
	for _, part := range strings.FieldsFunc(osmRef, func(r rune) bool {
		return r == ';' || r == '/' || r == ','
	}) {
		if normalizePlatformRef(part) == normTarget {
			return true
		}
	}
	return false
}

func normalizePlatformRef(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "voie ")
	s = strings.TrimPrefix(s, "quai ")
	return strings.TrimSpace(s)
}

func ComputeExitSide(fromLat, fromLon, stopLat, stopLon, platLat, platLon float64) string {
	return computeExitSideWithDir(stopLat, stopLon, stopLat-fromLat, stopLon-fromLon, platLat, platLon)
}

func computeExitSideWithDir(stopLat, stopLon, dirLat, dirLon, platLat, platLon float64) string {
	cosLat := math.Cos(stopLat * math.Pi / 180.0)
	dx := dirLon * cosLat
	dy := dirLat
	if dx == 0 && dy == 0 {
		return ""
	}

	px := (platLon - stopLon) * cosLat
	py := platLat - stopLat
	if px == 0 && py == 0 {
		return ""
	}

	cross := dx*py - dy*px
	if cross > 0 {
		return "left"
	}
	if cross < 0 {
		return "right"
	}
	return ""
}
