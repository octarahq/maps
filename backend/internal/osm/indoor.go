package osm

import (
	"container/heap"
	"context"
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"maps/backend/internal/utils"

	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
)

type IndoorNode struct {
	Lat   float32
	Lon   float32
	Level string
	Kind  string
	Name  string
}

type IndoorEdge struct {
	To         int32
	DistM      float32
	Mode       string
	FromLevel  string
	ToLevel    string
	Name       string
	Wheelchair bool
}

type IndoorGraph struct {
	Nodes []IndoorNode
	Adj   [][]IndoorEdge
}

type IndoorStep struct {
	Instruction string  `json:"instruction"`
	Kind        string  `json:"kind"`
	DistanceM   int     `json:"distance_m"`
	DurationSec int     `json:"duration_sec"`
	FromLevel   string  `json:"from_level,omitempty"`
	ToLevel     string  `json:"to_level,omitempty"`
	BearingDeg  float64 `json:"bearing_deg"`
	Turn        string  `json:"turn"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	EndLat      float64 `json:"end_lat"`
	EndLon      float64 `json:"end_lon"`
}

type IndoorRouteResult struct {
	Found            bool         `json:"found"`
	TotalDistanceM   int          `json:"total_distance_m"`
	TotalDurationSec int          `json:"total_duration_sec"`
	Steps            []IndoorStep `json:"steps"`
}

const defaultIndoorCachePath = "data/processed/osm_indoor_v2.gob"

var (
	indoorOnce  sync.Once
	indoorGraph *IndoorGraph
	indoorGrid  map[gridKey][]int32
)

func EnsureIndoorLoaded() {
	indoorOnce.Do(func() {
		if g, err := loadIndoorCache(defaultIndoorCachePath); err == nil && len(g.Nodes) > 0 {
			indoorGraph = g
			buildIndoorGrid(g)
			return
		}
		g, err := parsePBFIndoorGraph(defaultPBFPath)
		if err != nil {
			indoorGraph = &IndoorGraph{}
			indoorGrid = make(map[gridKey][]int32)
			return
		}
		_ = saveIndoorCache(defaultIndoorCachePath, g)
		indoorGraph = g
		buildIndoorGrid(g)
		runtime.GC()
		debug.FreeOSMemory()
	})
}

func buildIndoorGrid(g *IndoorGraph) {
	gridMap := make(map[gridKey][]int32, len(g.Nodes)/8)
	for i, n := range g.Nodes {
		k := gridKey{
			LatCell: int(math.Floor(float64(n.Lat) * 100)),
			LonCell: int(math.Floor(float64(n.Lon) * 100)),
		}
		gridMap[k] = append(gridMap[k], int32(i))
	}
	indoorGrid = gridMap
}

func loadIndoorCache(path string) (*IndoorGraph, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var g IndoorGraph
	if err := gob.NewDecoder(f).Decode(&g); err != nil {
		return nil, err
	}
	return &g, nil
}

func saveIndoorCache(path string, g *IndoorGraph) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(g)
}

type rawIndoorWay struct {
	nodes      []osm.NodeID
	mode       string
	level      string
	name       string
	wheelchair bool
}

type rawIndoorNodeMeta struct {
	lat   float32
	lon   float32
	level string
	kind  string
	name  string
}

func parsePBFIndoorGraph(path string) (*IndoorGraph, error) {
	neededNodes := make(map[osm.NodeID]struct{}, 250_000)
	var ways []rawIndoorWay
	var specialNodes []osm.NodeID

	f1, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	s1 := osmpbf.New(context.Background(), f1, 4)
	for s1.Scan() {
		switch o := s1.Object().(type) {
		case *osm.Node:
			if isSpecialIndoorNode(o.Tags) {
				neededNodes[o.ID] = struct{}{}
				specialNodes = append(specialNodes, o.ID)
			}
		case *osm.Way:
			mode, ok := classifyIndoorWay(o.Tags)
			if !ok || len(o.Nodes) < 2 {
				continue
			}
			ids := make([]osm.NodeID, len(o.Nodes))
			for i, wn := range o.Nodes {
				ids[i] = wn.ID
				neededNodes[wn.ID] = struct{}{}
			}
			ways = append(ways, rawIndoorWay{
				nodes:      ids,
				mode:       mode,
				level:      strings.TrimSpace(o.Tags.Find("level")),
				name:       strings.TrimSpace(o.Tags.Find("name")),
				wheelchair: wayAllowsWheelchair(o.Tags, mode),
			})
		}
	}
	err1 := s1.Err()
	s1.Close()
	f1.Close()
	if err1 != nil {
		return nil, err1
	}

	nodeMeta := make(map[osm.NodeID]rawIndoorNodeMeta, len(neededNodes))
	if len(neededNodes) > 0 {
		f2, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		s2 := osmpbf.New(context.Background(), f2, 4)
		for s2.Scan() {
			if n, ok := s2.Object().(*osm.Node); ok {
				if _, needed := neededNodes[n.ID]; needed {
					nodeMeta[n.ID] = rawIndoorNodeMeta{
						lat:   float32(n.Lat),
						lon:   float32(n.Lon),
						level: strings.TrimSpace(n.Tags.Find("level")),
						kind:  classifyIndoorNodeKind(n.Tags),
						name:  strings.TrimSpace(n.Tags.Find("name")),
					}
				}
			}
		}
		err2 := s2.Err()
		s2.Close()
		f2.Close()
		if err2 != nil {
			return nil, err2
		}
	}
	neededNodes = nil

	g := &IndoorGraph{
		Nodes: make([]IndoorNode, 0, len(nodeMeta)),
	}
	osmToIdx := make(map[osm.NodeID]int32, len(nodeMeta))

	getOrCreateNode := func(id osm.NodeID, fallbackLevel string) (int32, bool) {
		if idx, exists := osmToIdx[id]; exists {
			return idx, true
		}
		meta, ok := nodeMeta[id]
		if !ok {
			return 0, false
		}
		lvl := meta.level
		if lvl == "" {
			lvl = fallbackLevel
		}
		idx := int32(len(g.Nodes))
		g.Nodes = append(g.Nodes, IndoorNode{
			Lat:   meta.lat,
			Lon:   meta.lon,
			Level: lvl,
			Kind:  meta.kind,
			Name:  meta.name,
		})
		g.Adj = append(g.Adj, nil)
		osmToIdx[id] = idx
		return idx, true
	}

	addEdge := func(u, v int32, distM float32, mode, fromLvl, toLvl, name string, wheelchair bool) {
		g.Adj[u] = append(g.Adj[u], IndoorEdge{
			To:         v,
			DistM:      distM,
			Mode:       mode,
			FromLevel:  fromLvl,
			ToLevel:    toLvl,
			Name:       name,
			Wheelchair: wheelchair,
		})
	}

	for _, w := range ways {
		for i := 0; i < len(w.nodes)-1; i++ {
			u, okU := getOrCreateNode(w.nodes[i], w.level)
			v, okV := getOrCreateNode(w.nodes[i+1], w.level)
			if !okU || !okV || u == v {
				continue
			}
			nu, nv := g.Nodes[u], g.Nodes[v]
			distM := float32(utils.CoordsGetDistance(float64(nu.Lon), float64(nu.Lat), float64(nv.Lon), float64(nv.Lat)) * 1000.0)
			if distM < 0.5 {
				distM = 0.5
			}
			fromLvl := firstLevelToken(nu.Level, w.level)
			toLvl := firstLevelToken(nv.Level, w.level)
			if w.mode == "escalator" || w.mode == "stairs" || w.mode == "elevator" {
				if l1, l2, ok := splitLevelRange(w.level); ok {
					if fromLvl == "" || fromLvl == w.level {
						fromLvl = l1
					}
					if toLvl == "" || toLvl == w.level {
						toLvl = l2
					}
				}
			}
			edgeMode := w.mode
			if nu.Kind == "gate" || nv.Kind == "gate" {
				edgeMode = "gate"
			}
			addEdge(u, v, distM, edgeMode, fromLvl, toLvl, w.name, w.wheelchair)
			addEdge(v, u, distM, edgeMode, toLvl, fromLvl, w.name, w.wheelchair)
		}
	}

	for _, id := range specialNodes {
		_, _ = getOrCreateNode(id, "")
	}

	tempGrid := make(map[gridKey][]int32, len(g.Nodes)/8)
	for i, n := range g.Nodes {
		k := gridKey{
			LatCell: int(math.Floor(float64(n.Lat) * 1000)),
			LonCell: int(math.Floor(float64(n.Lon) * 1000)),
		}
		tempGrid[k] = append(tempGrid[k], int32(i))
	}
	for i, n := range g.Nodes {
		cLat := int(math.Floor(float64(n.Lat) * 1000))
		cLon := int(math.Floor(float64(n.Lon) * 1000))
		for dLat := -1; dLat <= 1; dLat++ {
			for dLon := -1; dLon <= 1; dLon++ {
				for _, j := range tempGrid[gridKey{cLat + dLat, cLon + dLon}] {
					if int32(i) >= j {
						continue
					}
					nj := g.Nodes[j]
					d := float32(utils.CoordsGetDistance(float64(n.Lon), float64(n.Lat), float64(nj.Lon), float64(nj.Lat)) * 1000.0)
					if d > 8.0 {
						continue
					}
					isElev := n.Kind == "elevator" || nj.Kind == "elevator"
					sameLevel := n.Level == "" || nj.Level == "" || levelsOverlap(n.Level, nj.Level)
					if !isElev && !sameLevel {
						continue
					}
					mode := "walk"
					if isElev && !sameLevel {
						mode = "elevator"
					}
					addEdge(int32(i), j, max(d, 1.0), mode, firstLevelToken(n.Level, ""), firstLevelToken(nj.Level, ""), "", true)
					addEdge(j, int32(i), max(d, 1.0), mode, firstLevelToken(nj.Level, ""), firstLevelToken(n.Level, ""), "", true)
				}
			}
		}
	}

	ways = nil
	nodeMeta = nil
	osmToIdx = nil
	specialNodes = nil
	tempGrid = nil

	return g, nil
}

func isSpecialIndoorNode(tags osm.Tags) bool {
	if tags.Find("highway") == "elevator" || tags.Find("railway") == "subway_entrance" {
		return true
	}
	b := tags.Find("barrier")
	return b == "turnstile" || b == "ticket_gate"
}

func classifyIndoorNodeKind(tags osm.Tags) string {
	if tags.Find("highway") == "elevator" {
		return "elevator"
	}
	b := tags.Find("barrier")
	if b == "turnstile" || b == "ticket_gate" {
		return "gate"
	}
	if tags.Find("railway") == "subway_entrance" {
		return "entrance"
	}
	if tags.Find("public_transport") == "platform" || tags.Find("railway") == "platform" {
		return "platform"
	}
	return ""
}

func classifyIndoorWay(tags osm.Tags) (string, bool) {
	if tags.Find("highway") == "elevator" {
		return "elevator", true
	}
	conv := tags.Find("conveying")
	if conv != "" && conv != "no" {
		return "escalator", true
	}
	hw := tags.Find("highway")
	if hw == "steps" {
		return "stairs", true
	}
	if tags.Find("railway") == "platform" || tags.Find("public_transport") == "platform" {
		if tags.Find("bus") == "yes" && tags.Find("subway") != "yes" && tags.Find("train") != "yes" && tags.Find("tram") != "yes" {
			return "", false
		}
		return "walk", true
	}
	if hw == "corridor" {
		return "walk", true
	}
	if hw == "footway" || hw == "pedestrian" || hw == "path" {
		lvl := tags.Find("level")
		ind := tags.Find("indoor")
		tun := tags.Find("tunnel")
		lay := tags.Find("layer")
		cov := tags.Find("covered")
		if lvl != "" || (ind != "" && ind != "no") || tun == "yes" || cov == "yes" || strings.HasPrefix(lay, "-") {
			return "walk", true
		}
	}
	return "", false
}

func wayAllowsWheelchair(tags osm.Tags, mode string) bool {
	w := strings.ToLower(strings.TrimSpace(tags.Find("wheelchair")))
	if w == "no" {
		return false
	}
	if w == "yes" || w == "designated" {
		return true
	}
	return mode != "stairs" && mode != "escalator"
}

func firstLevelToken(a, fallback string) string {
	s := a
	if s == "" {
		s = fallback
	}
	if s == "" {
		return ""
	}
	if idx := strings.IndexAny(s, ";,"); idx != -1 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}

func splitLevelRange(lvl string) (string, string, bool) {
	parts := strings.FieldsFunc(lvl, func(r rune) bool {
		return r == ';' || r == ','
	})
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[len(parts)-1]), true
	}
	return "", "", false
}

func levelsOverlap(a, b string) bool {
	pa := strings.FieldsFunc(a, func(r rune) bool { return r == ';' || r == ',' })
	pb := strings.FieldsFunc(b, func(r rune) bool { return r == ';' || r == ',' })
	for _, x := range pa {
		tx := strings.TrimSpace(x)
		for _, y := range pb {
			if tx != "" && tx == strings.TrimSpace(y) {
				return true
			}
		}
	}
	return false
}

type astarItem struct {
	node int32
	f    float64
	g    float64
	idx  int
}

type astarHeap []*astarItem

func (h astarHeap) Len() int           { return len(h) }
func (h astarHeap) Less(i, j int) bool { return h[i].f < h[j].f }
func (h astarHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i]; h[i].idx = i; h[j].idx = j }
func (h *astarHeap) Push(x any)        { it := x.(*astarItem); it.idx = len(*h); *h = append(*h, it) }
func (h *astarHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]
	return it
}

func findNearestIndoorCandidates(lat, lon float64, preferredLevel string, maxRadiusM float64, limit int) []struct {
	idx  int32
	cost float64
} {
	if indoorGraph == nil || len(indoorGraph.Nodes) == 0 {
		return nil
	}
	cLat := int(math.Floor(lat * 100))
	cLon := int(math.Floor(lon * 100))

	type cand struct {
		idx  int32
		cost float64
	}
	var cands []cand
	for dLat := -1; dLat <= 1; dLat++ {
		for dLon := -1; dLon <= 1; dLon++ {
			for _, idx := range indoorGrid[gridKey{cLat + dLat, cLon + dLon}] {
				if len(indoorGraph.Adj[idx]) == 0 {
					continue
				}
				n := indoorGraph.Nodes[idx]
				d := utils.CoordsGetDistance(lon, lat, float64(n.Lon), float64(n.Lat)) * 1000.0
				if d > maxRadiusM {
					continue
				}
				score := d
				if preferredLevel != "" && levelsOverlap(n.Level, preferredLevel) {
					score -= 35.0
				}
				cands = append(cands, cand{idx: idx, cost: score})
			}
		}
	}

	res := make([]struct {
		idx  int32
		cost float64
	}, 0, min(limit, len(cands)))
	for i := 0; i < len(cands) && i < limit; i++ {
		best := i
		for j := i + 1; j < len(cands); j++ {
			if cands[j].cost < cands[best].cost {
				best = j
			}
		}
		cands[i], cands[best] = cands[best], cands[i]
		res = append(res, struct {
			idx  int32
			cost float64
		}{idx: cands[i].idx, cost: max(0.0, cands[i].cost)})
	}
	return res
}

func FindIndoorRoute(fromLat, fromLon float64, fromLevel string, toLat, toLon float64, toLevel string, wheelchair bool) IndoorRouteResult {
	EnsureIndoorLoaded()
	startCands := findNearestIndoorCandidates(fromLat, fromLon, fromLevel, 250, 24)
	goalCands := findNearestIndoorCandidates(toLat, toLon, toLevel, 250, 24)
	if len(startCands) == 0 || len(goalCands) == 0 {
		return IndoorRouteResult{Found: false}
	}

	goalSet := make(map[int32]bool, len(goalCands))
	for _, g := range goalCands {
		goalSet[g.idx] = true
	}

	gScore := make(map[int32]float64, 512)
	parent := make(map[int32]int32, 512)
	parentEdge := make(map[int32]IndoorEdge, 512)
	closed := make(map[int32]bool, 512)
	startSet := make(map[int32]bool, len(startCands))

	heuristic := func(idx int32) float64 {
		n := indoorGraph.Nodes[idx]
		return utils.CoordsGetDistance(float64(n.Lon), float64(n.Lat), toLon, toLat) * 1000.0
	}

	pq := &astarHeap{}
	heap.Init(pq)
	for _, s := range startCands {
		if goalSet[s.idx] {
			continue
		}
		startSet[s.idx] = true
		gScore[s.idx] = s.cost
		heap.Push(pq, &astarItem{node: s.idx, g: s.cost, f: s.cost + heuristic(s.idx)})
	}

	var reachedGoal int32 = -1
	var closestVisited int32 = -1
	closestVisitedDist := math.MaxFloat64
	visited := 0
	const maxVisited = 20000

	for pq.Len() > 0 && visited < maxVisited {
		cur := heap.Pop(pq).(*astarItem)
		if goalSet[cur.node] {
			reachedGoal = cur.node
			break
		}
		if closed[cur.node] {
			continue
		}
		closed[cur.node] = true
		visited++

		if dToGoal := heuristic(cur.node); dToGoal < closestVisitedDist {
			closestVisitedDist = dToGoal
			closestVisited = cur.node
		}

		for _, e := range indoorGraph.Adj[cur.node] {
			if closed[e.To] {
				continue
			}
			if wheelchair && !e.Wheelchair {
				continue
			}
			cost := float64(e.DistM)
			switch e.Mode {
			case "stairs":
				cost = cost*1.8 + 8.0
			case "escalator":
				cost = cost*1.2 + 4.0
			case "elevator":
				if wheelchair {
					cost += 5.0
				} else {
					cost = cost*1.5 + 15.0
				}
			case "gate":
				cost += 5.0
			}

			tentativeG := cur.g + cost
			if prevG, ok := gScore[e.To]; !ok || tentativeG < prevG {
				gScore[e.To] = tentativeG
				parent[e.To] = cur.node
				parentEdge[e.To] = e
				heap.Push(pq, &astarItem{
					node: e.To,
					g:    tentativeG,
					f:    tentativeG + heuristic(e.To),
				})
			}
		}
	}

	directDistM := utils.CoordsGetDistance(fromLon, fromLat, toLon, toLat) * 1000.0

	needsBridge := false
	if reachedGoal < 0 {

		if closestVisited >= 0 && closestVisitedDist <= 220.0 && closestVisitedDist < directDistM {
			reachedGoal = closestVisited
			needsBridge = true
		} else if directDistM >= 5.0 && directDistM <= 600.0 {
			distInt := max(1, int(math.Round(directDistM)))
			dur := max(5, int(math.Round(directDistM/1.15)))
			b := math.Round(bearingDegrees(fromLat, fromLon, toLat, toLon)*10) / 10
			return IndoorRouteResult{
				Found:            true,
				TotalDistanceM:   distInt,
				TotalDurationSec: dur,
				Steps: []IndoorStep{
					{
						Instruction: formatIndoorInstruction("walk", "straight", fromLevel, toLevel, "couloir de correspondance"),
						Kind:        "walk",
						DistanceM:   distInt,
						DurationSec: dur,
						FromLevel:   fromLevel,
						ToLevel:     toLevel,
						BearingDeg:  b,
						Turn:        "straight",
						Lat:         fromLat,
						Lon:         fromLon,
						EndLat:      toLat,
						EndLon:      toLon,
					},
				},
			}
		} else {
			return IndoorRouteResult{Found: false}
		}
	}

	type rawSeg struct {
		from IndoorNode
		to   IndoorNode
		edge IndoorEdge
	}
	var rev []rawSeg
	cur := reachedGoal
	for !startSet[cur] {
		p, ok := parent[cur]
		if !ok {
			break
		}
		e := parentEdge[cur]
		rev = append(rev, rawSeg{
			from: indoorGraph.Nodes[p],
			to:   indoorGraph.Nodes[cur],
			edge: e,
		})
		cur = p
	}
	if len(rev) == 0 && !needsBridge {
		return IndoorRouteResult{Found: false}
	}

	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}

	if needsBridge {
		lastNode := indoorGraph.Nodes[reachedGoal]
		bridgeDist := float32(utils.CoordsGetDistance(float64(lastNode.Lon), float64(lastNode.Lat), toLon, toLat) * 1000.0)
		if bridgeDist >= 3.0 {
			rev = append(rev, rawSeg{
				from: lastNode,
				to: IndoorNode{
					Lat:   float32(toLat),
					Lon:   float32(toLon),
					Level: toLevel,
				},
				edge: IndoorEdge{
					DistM:      bridgeDist,
					Mode:       "walk",
					FromLevel:  firstLevelToken(lastNode.Level, ""),
					ToLevel:    toLevel,
					Name:       "hall de correspondance",
					Wheelchair: true,
				},
			})
		}
	}

	type mergedStep struct {
		mode      string
		fromLevel string
		toLevel   string
		name      string
		distM     float64
		startLat  float64
		startLon  float64
		endLat    float64
		endLon    float64
		bearing   float64
	}

	var merged []mergedStep
	for _, seg := range rev {
		b := bearingDegrees(float64(seg.from.Lat), float64(seg.from.Lon), float64(seg.to.Lat), float64(seg.to.Lon))
		if len(merged) > 0 {
			last := &merged[len(merged)-1]
			sameMode := last.mode == seg.edge.Mode
			sameLevels := last.fromLevel == seg.edge.FromLevel && last.toLevel == seg.edge.ToLevel
			smallTurn := angleDiffDeg(last.bearing, b) < 42.0
			tinyMicroSeg := seg.edge.Mode == "walk" && last.mode == "walk" && (seg.edge.DistM < 5.0 || last.distM < 5.0)
			if (sameMode && sameLevels && (seg.edge.Mode != "walk" || smallTurn)) || tinyMicroSeg {
				last.distM += float64(seg.edge.DistM)
				last.endLat = float64(seg.to.Lat)
				last.endLon = float64(seg.to.Lon)
				if seg.edge.ToLevel != "" {
					last.toLevel = seg.edge.ToLevel
				}
				last.bearing = bearingDegrees(last.startLat, last.startLon, last.endLat, last.endLon)
				if last.name == "" {
					last.name = seg.edge.Name
				}
				continue
			}
		}
		merged = append(merged, mergedStep{
			mode:      seg.edge.Mode,
			fromLevel: seg.edge.FromLevel,
			toLevel:   seg.edge.ToLevel,
			name:      seg.edge.Name,
			distM:     float64(seg.edge.DistM),
			startLat:  float64(seg.from.Lat),
			startLon:  float64(seg.from.Lon),
			endLat:    float64(seg.to.Lat),
			endLon:    float64(seg.to.Lon),
			bearing:   b,
		})
	}

	steps := make([]IndoorStep, 0, len(merged))
	totalDist := 0
	totalDur := 0
	for i, m := range merged {
		distInt := max(1, int(math.Round(m.distM)))
		speedMps := 1.15
		extraSec := 0
		switch m.mode {
		case "stairs":
			speedMps = 0.75
			extraSec = 5
		case "escalator":
			speedMps = 0.9
			extraSec = 4
		case "elevator":
			speedMps = 0.8
			extraSec = 18
		case "gate":
			extraSec = 5
		}
		dur := max(3, int(math.Round(m.distM/speedMps))+extraSec)

		turn := "straight"
		if i > 0 && m.mode == "walk" {
			turn = classifyTurn(merged[i-1].bearing, m.bearing)
		}

		instr := formatIndoorInstruction(m.mode, turn, m.fromLevel, m.toLevel, m.name)
		steps = append(steps, IndoorStep{
			Instruction: instr,
			Kind:        m.mode,
			DistanceM:   distInt,
			DurationSec: dur,
			FromLevel:   m.fromLevel,
			ToLevel:     m.toLevel,
			BearingDeg:  math.Round(m.bearing*10) / 10,
			Turn:        turn,
			Lat:         m.startLat,
			Lon:         m.startLon,
			EndLat:      m.endLat,
			EndLon:      m.endLon,
		})
		totalDist += distInt
		totalDur += dur
	}

	return IndoorRouteResult{
		Found:            true,
		TotalDistanceM:   totalDist,
		TotalDurationSec: totalDur,
		Steps:            steps,
	}
}

func bearingDegrees(lat1, lon1, lat2, lon2 float64) float64 {
	phi1 := lat1 * math.Pi / 180.0
	phi2 := lat2 * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	y := math.Sin(dLon) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(dLon)
	deg := math.Atan2(y, x) * 180.0 / math.Pi
	return math.Mod(deg+360.0, 360.0)
}

func angleDiffDeg(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func classifyTurn(prevBearing, newBearing float64) string {
	delta := math.Mod(newBearing-prevBearing+540.0, 360.0) - 180.0
	switch {
	case delta >= -25 && delta <= 25:
		return "straight"
	case delta > 25 && delta <= 60:
		return "slight_right"
	case delta > 60 && delta <= 125:
		return "right"
	case delta > 125:
		return "sharp_right"
	case delta < -25 && delta >= -60:
		return "slight_left"
	case delta < -60 && delta >= -125:
		return "left"
	default:
		return "sharp_left"
	}
}

func formatIndoorInstruction(mode, turn, fromLvl, toLvl, name string) string {
	lvlSuffix := ""
	if fromLvl != "" && toLvl != "" && fromLvl != toLvl {
		lvlSuffix = fmt.Sprintf(" (niveau %s → %s)", fromLvl, toLvl)
	} else if toLvl != "" {
		lvlSuffix = fmt.Sprintf(" (niveau %s)", toLvl)
	}

	dirVerb := "Prenez"
	if fNum, errF := strconv.ParseFloat(fromLvl, 64); errF == nil {
		if tNum, errT := strconv.ParseFloat(toLvl, 64); errT == nil {
			if tNum > fNum {
				dirVerb = "Montez"
			} else if tNum < fNum {
				dirVerb = "Descendez"
			}
		}
	}

	switch mode {
	case "stairs":
		return fmt.Sprintf("%s les escaliers%s", dirVerb, lvlSuffix)
	case "escalator":
		if dirVerb == "Prenez" {
			return fmt.Sprintf("Prenez l'escalator%s", lvlSuffix)
		}
		return fmt.Sprintf("%s par l'escalator%s", dirVerb, lvlSuffix)
	case "elevator":
		return fmt.Sprintf("Prenez l'ascenseur%s", lvlSuffix)
	case "gate":
		return "Passez les portillons de validation"
	default:
		turnPrefix := map[string]string{
			"straight":     "Continuez tout droit",
			"slight_right": "Serrez légèrement à droite",
			"right":        "Tournez à droite",
			"sharp_right":  "Tournez franchement à droite",
			"slight_left":  "Serrez légèrement à gauche",
			"left":         "Tournez à gauche",
			"sharp_left":   "Tournez franchement à gauche",
		}[turn]
		if turnPrefix == "" {
			turnPrefix = "Continuez tout droit"
		}
		if name != "" {
			return fmt.Sprintf("%s (%s)%s", turnPrefix, name, lvlSuffix)
		}
		return fmt.Sprintf("%s dans le couloir%s", turnPrefix, lvlSuffix)
	}
}
