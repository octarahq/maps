package routes

import (
	"cmp"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"

	"maps/backend/internal/engine"
	"maps/backend/internal/types"
	"maps/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type BBox struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

func tileToBBox(x, y, z int) BBox {
	n := math.Pow(2.0, float64(z))
	lonMin := float64(x)/n*360.0 - 180.0
	lonMax := float64(x+1)/n*360.0 - 180.0

	latMinRad := math.Atan(math.Sinh(math.Pi * (1.0 - 2.0*float64(y+1)/n)))
	latMaxRad := math.Atan(math.Sinh(math.Pi * (1.0 - 2.0*float64(y)/n)))
	latMin := latMinRad * 180.0 / math.Pi
	latMax := latMaxRad * 180.0 / math.Pi

	if latMin > latMax {
		latMin, latMax = latMax, latMin
	}
	if lonMin > lonMax {
		lonMin, lonMax = lonMax, lonMin
	}

	return BBox{
		MinLat: latMin,
		MaxLat: latMax,
		MinLon: lonMin,
		MaxLon: lonMax,
	}
}

type RouteShapeEntry struct {
	RouteID string
	Color   string
	Points  [][2]float64
	MinLat  float64
	MaxLat  float64
	MinLon  float64
	MaxLon  float64
}

type gridCell struct {
	x, y int32
}

const branchCellStep = 0.0004

func toGridCell(p [2]float64) gridCell {
	return gridCell{
		x: int32(math.Floor(p[0] / branchCellStep)),
		y: int32(math.Floor(p[1] / branchCellStep)),
	}
}

func extractUniqueRouteBranches(shapes [][][2]float64) [][][2]float64 {
	slices.SortFunc(shapes, func(a, b [][2]float64) int {
		return cmp.Compare(len(b), len(a))
	})

	covered := make(map[gridCell]bool)
	markPoint := func(p [2]float64) {
		c := toGridCell(p)
		for dx := int32(-1); dx <= 1; dx++ {
			for dy := int32(-1); dy <= 1; dy++ {
				covered[gridCell{c.x + dx, c.y + dy}] = true
			}
		}
	}
	markCovered := func(pts [][2]float64) {
		for i, p := range pts {
			markPoint(p)
			if i+1 < len(pts) {
				p2 := pts[i+1]
				dLat := p2[0] - p[0]
				dLon := p2[1] - p[1]
				dist := math.Hypot(dLat, dLon)
				steps := int(dist / (branchCellStep * 0.5))
				for s := 1; s < steps; s++ {
					t := float64(s) / float64(steps)
					markPoint([2]float64{p[0] + dLat*t, p[1] + dLon*t})
				}
			}
		}
	}

	var result [][][2]float64
	for _, pts := range shapes {
		n := len(pts)
		if n < 2 {
			continue
		}
		if len(result) == 0 {
			result = append(result, pts)
			markCovered(pts)
			continue
		}

		isUncovered := make([]bool, n)
		uncoveredCount := 0
		for i, p := range pts {
			if !covered[toGridCell(p)] {
				isUncovered[i] = true
				uncoveredCount++
			}
		}
		if uncoveredCount == 0 {
			continue
		}

		lastUncov := -1
		for i := 0; i < n; i++ {
			if isUncovered[i] {
				if lastUncov >= 0 && i-lastUncov <= 5 {
					for j := lastUncov + 1; j < i; j++ {
						isUncovered[j] = true
					}
				}
				lastUncov = i
			}
		}

		i := 0
		var addedBranches [][][2]float64
		for i < n {
			if !isUncovered[i] {
				i++
				continue
			}
			start := i
			for i < n && isUncovered[i] {
				i++
			}
			end := i - 1

			branchStart := start
			if branchStart > 0 {
				branchStart--
			}
			branchEnd := end
			if branchEnd < n-1 {
				branchEnd++
			}

			if branchEnd-branchStart >= 1 {
				branch := pts[branchStart : branchEnd+1]
				addedBranches = append(addedBranches, branch)
			}
		}

		for _, b := range addedBranches {
			result = append(result, b)
			markCovered(b)
		}
	}

	return result
}

func clipPolylineToBBox(pts [][2]float64, bbox BBox) [][][2]float64 {
	latMargin := math.Max(0.015, (bbox.MaxLat-bbox.MinLat)*0.25)
	lonMargin := math.Max(0.015, (bbox.MaxLon-bbox.MinLon)*0.25)
	minLat := bbox.MinLat - latMargin
	maxLat := bbox.MaxLat + latMargin
	minLon := bbox.MinLon - lonMargin
	maxLon := bbox.MaxLon + lonMargin

	var segments [][][2]float64
	var current [][2]float64

	for i := 0; i < len(pts)-1; i++ {
		p1 := pts[i]
		p2 := pts[i+1]

		segMinLat, segMaxLat := p1[0], p2[0]
		if segMinLat > segMaxLat {
			segMinLat, segMaxLat = segMaxLat, segMinLat
		}
		segMinLon, segMaxLon := p1[1], p2[1]
		if segMinLon > segMaxLon {
			segMinLon, segMaxLon = segMaxLon, segMinLon
		}

		intersects := segMaxLat >= minLat && segMinLat <= maxLat &&
			segMaxLon >= minLon && segMinLon <= maxLon

		if intersects {
			if len(current) == 0 {
				current = append(current, p1)
			}
			current = append(current, p2)
		} else {
			if len(current) >= 2 {
				segments = append(segments, current)
			}
			current = nil
		}
	}
	if len(current) >= 2 {
		segments = append(segments, current)
	}
	return segments
}

var (
	routeShapesOnce sync.Once
	routeShapesList []RouteShapeEntry

	layerStopsOnce    sync.Once
	detailedStopsList []types.ParentsStation
)

func isRailRoute(r types.Route, shapeID string) bool {
	if r.RType == types.RouteTypeTrain || r.RType == types.RouteTypeMetro ||
		r.RType == types.RouteTypeTram || r.RType == types.RouteTypeFunicular ||
		r.RType == types.RouteTypeCableTram || r.RType == types.RouteTypeAerialLift {
		return true
	}
	parts := strings.Split(shapeID, "_")
	if len(parts) >= 2 {
		switch parts[1] {
		case "0", "1", "2", "6", "7":
			return true
		}
	}
	return false
}

func initRouteShapes(e *engine.Engine) {
	routeShapesOnce.Do(func() {
		shapesIdx := e.GetShapesIndex()
		if len(shapesIdx) == 0 {
			return
		}

		routesMap := utils.IndexRoutesByID(e.Net.Routes)

		shapeToRoute := make(map[string]types.Route, len(e.Net.Trips))
		for _, trip := range e.Net.Trips {
			if trip.ShapeID == "" {
				continue
			}
			cleanShp := utils.IdRemoveIDFMPrefix(trip.ShapeID)
			if _, exists := shapeToRoute[cleanShp]; exists {
				continue
			}
			r, ok := routesMap[trip.RID]
			if !ok {
				r, ok = routesMap[utils.IdRemoveIDFMPrefix(trip.RID)]
			}
			if ok {
				shapeToRoute[trip.ShapeID] = r
				shapeToRoute[cleanShp] = r
			}
		}

		type routeRawShapes struct {
			route  types.Route
			color  string
			shapes [][][2]float64
		}
		byRoute := make(map[string]*routeRawShapes)
		seenCleanShape := make(map[string]bool, len(shapesIdx))

		for shapeID, pts := range shapesIdx {
			if len(pts) < 2 {
				continue
			}
			cleanShp := utils.IdRemoveIDFMPrefix(shapeID)
			if seenCleanShape[cleanShp] {
				continue
			}

			r, ok := shapeToRoute[shapeID]
			if !ok {
				r, ok = shapeToRoute[cleanShp]
			}
			if !ok || !isRailRoute(r, shapeID) {
				continue
			}
			seenCleanShape[cleanShp] = true

			cleanRID := utils.IdRemoveIDFMPrefix(r.RID)
			entry := byRoute[cleanRID]
			if entry == nil {
				color := r.Color
				if color == "" {
					color = "0071BD"
				}
				if !strings.HasPrefix(color, "#") {
					color = "#" + color
				}
				entry = &routeRawShapes{
					route: r,
					color: color,
				}
				byRoute[cleanRID] = entry
			}
			entry.shapes = append(entry.shapes, pts)
		}

		for cleanRID, entry := range byRoute {
			branches := extractUniqueRouteBranches(entry.shapes)
			for _, pts := range branches {
				if len(pts) < 2 {
					continue
				}
				minLat, maxLat := pts[0][0], pts[0][0]
				minLon, maxLon := pts[0][1], pts[0][1]
				for _, p := range pts {
					if p[0] < minLat {
						minLat = p[0]
					}
					if p[0] > maxLat {
						maxLat = p[0]
					}
					if p[1] < minLon {
						minLon = p[1]
					}
					if p[1] > maxLon {
						maxLon = p[1]
					}
				}
				routeShapesList = append(routeShapesList, RouteShapeEntry{
					RouteID: cleanRID,
					Color:   entry.color,
					Points:  pts,
					MinLat:  minLat,
					MaxLat:  maxLat,
					MinLon:  minLon,
					MaxLon:  maxLon,
				})
			}
		}
	})
}

func isNoctilienName(name string) bool {
	return len(name) >= 2 && (name[0] == 'N' || name[0] == 'n') && name[1] >= '0' && name[1] <= '9'
}

func sortParentStationRoutes(routes []types.ParentsStationRoute) {
	slices.SortFunc(routes, func(a, b types.ParentsStationRoute) int {
		prioA := getRouteTypePriority(a.RType)
		prioB := getRouteTypePriority(b.RType)
		if prioA != prioB {
			return cmp.Compare(prioB, prioA)
		}
		nocA := isNoctilienName(a.Name)
		nocB := isNoctilienName(b.Name)
		if nocA != nocB {
			if nocA {
				return 1
			}
			return -1
		}
		return strings.Compare(a.Name, b.Name)
	})
}

func getRouteTypePriority(rt types.RouteType) int {
	switch rt {
	case types.RouteTypeTrain:
		return 5
	case types.RouteTypeMetro:
		return 4
	case types.RouteTypeTram:
		return 3
	case types.RouteTypeAerialLift, types.RouteTypeCableTram, types.RouteTypeFunicular:
		return 2
	default:
		return 1
	}
}

func routeTypeToParentType(rt types.RouteType) types.ParentType {
	switch rt {
	case types.RouteTypeTrain:
		return types.ParentTypeRail
	case types.RouteTypeMetro:
		return types.ParentTypeMetro
	case types.RouteTypeTram:
		return types.ParentTypeStreetTram
	case types.RouteTypeAerialLift, types.RouteTypeCableTram, types.RouteTypeFunicular:
		return types.ParentTypeLift
	default:
		return types.ParentTypeStreetBus
	}
}

func initDetailedStops(e *engine.Engine) {
	layerStopsOnce.Do(func() {
		stopsByID := utils.IndexStopsByID(e.Net.Stops)
		routesByID := utils.IndexRoutesByID(e.Net.Routes)

		var fallbackStopRoutes map[string]map[string]types.Route
		if len(e.Graph) == 0 && len(e.Net.StopTimes) > 0 {
			fallbackStopRoutes = make(map[string]map[string]types.Route)
			tripToRoute := make(map[string]types.Route, len(e.Net.Trips))
			for _, tr := range e.Net.Trips {
				if r, ok := routesByID[tr.RID]; ok {
					tripToRoute[tr.TID] = r
				} else if r, ok := routesByID[utils.IdRemoveIDFMPrefix(tr.RID)]; ok {
					tripToRoute[tr.TID] = r
				}
			}
			for _, st := range e.Net.StopTimes {
				r, ok := tripToRoute[st.TID]
				if !ok {
					continue
				}
				cleanSID := utils.IdRemoveIDFMPrefix(st.SId)
				if fallbackStopRoutes[cleanSID] == nil {
					fallbackStopRoutes[cleanSID] = make(map[string]types.Route)
				}
				fallbackStopRoutes[cleanSID][utils.IdRemoveIDFMPrefix(r.RID)] = r
			}
		}

		getRoutesForChildStop := func(cid string) map[string]types.Route {
			cleanCID := utils.IdRemoveIDFMPrefix(cid)
			if fallbackStopRoutes != nil {
				return fallbackStopRoutes[cleanCID]
			}
			res := make(map[string]types.Route)
			for _, edge := range e.GetEdgesForStop(cleanCID) {
				if edge.RID == "" {
					continue
				}
				if r, ok := routesByID[edge.RID]; ok {
					res[utils.IdRemoveIDFMPrefix(r.RID)] = r
				} else if r, ok := routesByID[utils.IdRemoveIDFMPrefix(edge.RID)]; ok {
					res[utils.IdRemoveIDFMPrefix(r.RID)] = r
				}
			}
			for _, edge := range e.GetReverseEdgesForStop(cleanCID) {
				if edge.RID == "" {
					continue
				}
				if r, ok := routesByID[edge.RID]; ok {
					res[utils.IdRemoveIDFMPrefix(r.RID)] = r
				} else if r, ok := routesByID[utils.IdRemoveIDFMPrefix(edge.RID)]; ok {
					res[utils.IdRemoveIDFMPrefix(r.RID)] = r
				}
			}
			return res
		}

		type groupKey struct {
			pType types.ParentType
			name  string
		}
		type subGroup struct {
			id     string
			name   string
			pType  types.ParentType
			sumLat float64
			sumLon float64
			count  int
			routes map[string]types.Route
		}

		detailedStopsList = make([]types.ParentsStation, 0, len(e.Net.ParentsStations)*2)

		for _, ps := range e.Net.ParentsStations {
			if ps.Lat == 0 && ps.Lon == 0 {
				continue
			}
			if len(ps.ChildrenStops) <= 1 {
				detailedStopsList = append(detailedStopsList, ps)
				continue
			}

			groups := make(map[groupKey]*subGroup)
			for _, cid := range ps.ChildrenStops {
				st, ok := stopsByID[cid]
				if !ok {
					st, ok = stopsByID[utils.IdRemoveIDFMPrefix(cid)]
				}
				if !ok || (st.Lat == 0 && st.Long == 0) {
					continue
				}
				rMap := getRoutesForChildStop(cid)
				if len(rMap) == 0 {
					continue
				}

				bestPrio := 0
				pType := types.ParentTypeStreetBus
				for _, r := range rMap {
					prio := getRouteTypePriority(r.RType)
					if prio > bestPrio {
						bestPrio = prio
						pType = routeTypeToParentType(r.RType)
					}
				}

				stopName := strings.TrimSpace(st.Name)
				if stopName == "" {
					stopName = ps.Name
				}
				k := groupKey{pType: pType, name: stopName}
				g := groups[k]
				if g == nil {
					g = &subGroup{
						id:     utils.IdRemoveIDFMPrefix(cid),
						name:   stopName,
						pType:  pType,
						routes: make(map[string]types.Route),
					}
					groups[k] = g
				}
				g.sumLat += st.Lat
				g.sumLon += st.Long
				g.count++
				for rid, r := range rMap {
					g.routes[rid] = r
				}
			}

			if len(groups) == 0 {
				detailedStopsList = append(detailedStopsList, ps)
				continue
			}

			for _, g := range groups {
				var psRoutes []types.ParentsStationRoute
				for _, r := range g.routes {
					name := r.SName
					if name == "" {
						name = r.LName
					}

					psRoutes = append(psRoutes, types.ParentsStationRoute{
						RID:   utils.IdRemoveIDFMPrefix(r.RID),
						Name:  name,
						Color: r.Color,
						RType: r.RType,
						Picto: r.Picto,
					})
				}
				sortParentStationRoutes(psRoutes)
				detailedStopsList = append(detailedStopsList, types.ParentsStation{
					PID:    g.id,
					Name:   g.name,
					PType:  g.pType,
					Lat:    g.sumLat / float64(g.count),
					Lon:    g.sumLon / float64(g.count),
					Routes: psRoutes,
				})
			}
		}
	})
}

func getSimplificationToleranceSq(zoom int) float64 {
	switch {
	case zoom <= 9:
		return 0.0004 * 0.0004
	case zoom <= 11:
		return 0.00018 * 0.00018
	case zoom <= 13:
		return 0.00007 * 0.00007
	case zoom <= 15:
		return 0.00002 * 0.00002
	default:
		return 0.0
	}
}

func perpendicularDistanceSq(pt, start, end [2]float64) float64 {
	dx := end[1] - start[1]
	dy := end[0] - start[0]
	norm := dx*dx + dy*dy
	if norm == 0 {
		return (pt[0]-start[0])*(pt[0]-start[0]) + (pt[1]-start[1])*(pt[1]-start[1])
	}
	u := ((pt[1]-start[1])*dx + (pt[0]-start[0])*dy) / norm
	if u < 0 {
		u = 0
	} else if u > 1 {
		u = 1
	}
	px := start[1] + u*dx
	py := start[0] + u*dy
	return (pt[0]-py)*(pt[0]-py) + (pt[1]-px)*(pt[1]-px)
}

func simplifyPoints(points [][2]float64, tolSq float64) [][2]float64 {
	if tolSq <= 0 || len(points) <= 2 {
		return points
	}

	dmax := 0.0
	index := 0
	for i := 1; i < len(points)-1; i++ {
		d := perpendicularDistanceSq(points[i], points[0], points[len(points)-1])
		if d > dmax {
			index = i
			dmax = d
		}
	}

	if dmax > tolSq {
		rec1 := simplifyPoints(points[:index+1], tolSq)
		rec2 := simplifyPoints(points[index:], tolSq)
		res := make([][2]float64, 0, len(rec1)+len(rec2)-1)
		res = append(res, rec1[:len(rec1)-1]...)
		res = append(res, rec2...)
		return res
	}

	return [][2]float64{points[0], points[len(points)-1]}
}

type LayerRouteLine struct {
	RouteID string       `json:"route_id"`
	Color   string       `json:"color"`
	Points  [][2]float64 `json:"points"`
}

type LayerStopRouteInfo struct {
	Name  string          `json:"name"`
	Picto string          `json:"picto"`
	Color string          `json:"color,omitempty"`
	Type  types.RouteType `json:"type"`
}

type LayerStopItem struct {
	ID     string               `json:"id"`
	Name   string               `json:"name"`
	Type   types.ParentType     `json:"type"`
	Lat    float64              `json:"lat"`
	Lon    float64              `json:"lon"`
	Routes []LayerStopRouteInfo `json:"routes"`
}

func getStopPriority(ps types.ParentsStation) int {
	hasTrain := false
	hasMetro := false
	hasTram := false
	hasCable := false
	hasBus := false

	switch ps.PType {
	case types.ParentTypeRail:
		hasTrain = true
	case types.ParentTypeMetro:
		hasMetro = true
	case types.ParentTypeStreetTram:
		hasTram = true
	case types.ParentTypeLift:
		hasCable = true
	case types.ParentTypeStreetBus:
		hasBus = true
	}

	for _, r := range ps.Routes {
		switch r.RType {
		case types.RouteTypeTrain:
			hasTrain = true
		case types.RouteTypeMetro:
			hasMetro = true
		case types.RouteTypeTram:
			hasTram = true
		case types.RouteTypeAerialLift, types.RouteTypeCableTram, types.RouteTypeFunicular:
			hasCable = true
		case types.RouteTypeBus:
			hasBus = true
		}
	}

	if hasTrain {
		return 5
	}
	if hasMetro {
		return 4
	}
	if hasTram {
		return 3
	}
	if hasCable {
		return 2
	}
	if hasBus {
		return 1
	}
	return 0
}

func matchesZoomLevel(prio int, zoom int) bool {
	switch {
	case zoom < 11:
		return prio >= 5
	case zoom < 13:
		return prio >= 4
	case zoom < 14:
		return prio >= 2
	default:
		return prio >= 1
	}
}

func getProximityThreshold(zoom int) float64 {
	switch {
	case zoom < 10:
		return 0.8
	case zoom < 11:
		return 0.5
	case zoom < 12:
		return 0.35
	case zoom < 13:
		return 0.22
	case zoom < 14:
		return 0.14
	case zoom < 15:
		return 0.08
	case zoom < 16:
		return 0.04
	default:
		return 0.012
	}
}

func buildStopRoutes(ps types.ParentsStation, maxZoom int) []LayerStopRouteInfo {
	seenPictos := make(map[string]bool)
	seenTypeAndNames := make(map[string]bool)

	sortedRoutes := make([]types.ParentsStationRoute, len(ps.Routes))
	copy(sortedRoutes, ps.Routes)
	sortParentStationRoutes(sortedRoutes)

	routesInfo := make([]LayerStopRouteInfo, 0, len(sortedRoutes))

	for _, r := range sortedRoutes {
		rPrio := getRouteTypePriority(r.RType)
		if maxZoom > 0 && !matchesZoomLevel(rPrio, maxZoom) {
			continue
		}

		cleanPicto := strings.TrimSpace(r.Picto)
		cleanName := strings.TrimSpace(r.Name)

		if cleanPicto != "" {
			if !seenPictos[cleanPicto] {
				seenPictos[cleanPicto] = true
				routesInfo = append(routesInfo, LayerStopRouteInfo{
					Name:  cleanName,
					Picto: cleanPicto,
					Color: r.Color,
					Type:  r.RType,
				})
			}
		} else if cleanName != "" {
			key := strconv.Itoa(int(r.RType)) + ":" + cleanName
			if !seenTypeAndNames[key] {
				seenTypeAndNames[key] = true
				routesInfo = append(routesInfo, LayerStopRouteInfo{
					Name:  cleanName,
					Picto: "",
					Color: r.Color,
					Type:  r.RType,
				})
			}
		}
	}
	return routesInfo
}

func handleLayerRequest(c *gin.Context, e engine.Engine) {
	layer := c.Param("layer")
	if layer == "" {
		layer = c.Query("layer")
	}

	var bbox *BBox
	zoom := 0

	sX := c.Param("x")
	sY := c.Param("y")
	sZ := c.Param("z")

	if sX == "" {
		sX = c.Query("x")
	}
	if sY == "" {
		sY = c.Query("y")
	}
	if sZ == "" {
		sZ = c.Query("z")
	}

	if sX != "" && sY != "" && sZ != "" {
		x, errX := strconv.Atoi(sX)
		y, errY := strconv.Atoi(sY)
		z, errZ := strconv.Atoi(sZ)
		if errX == nil && errY == nil && errZ == nil && z >= 0 && z <= 24 {
			b := tileToBBox(x, y, z)
			bbox = &b
			zoom = z
		}
	}

	if bbox == nil {
		minLatStr := c.Query("min_lat")
		maxLatStr := c.Query("max_lat")
		minLonStr := c.Query("min_lon")
		maxLonStr := c.Query("max_lon")
		if minLatStr != "" && maxLatStr != "" && minLonStr != "" && maxLonStr != "" {
			minLat, err1 := strconv.ParseFloat(minLatStr, 64)
			maxLat, err2 := strconv.ParseFloat(maxLatStr, 64)
			minLon, err3 := strconv.ParseFloat(minLonStr, 64)
			maxLon, err4 := strconv.ParseFloat(maxLonStr, 64)
			if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
				if minLat > maxLat {
					minLat, maxLat = maxLat, minLat
				}
				if minLon > maxLon {
					minLon, maxLon = maxLon, minLon
				}
				bbox = &BBox{
					MinLat: minLat,
					MaxLat: maxLat,
					MinLon: minLon,
					MaxLon: maxLon,
				}
			}
		}
	}
	if sZoom := c.Query("zoom"); sZoom != "" && zoom == 0 {
		if zVal, err := strconv.Atoi(sZoom); err == nil {
			zoom = zVal
		}
	}

	limit := 120
	if sLimit := c.Query("limit"); sLimit != "" {
		if lVal, err := strconv.Atoi(sLimit); err == nil && lVal > 0 {
			limit = lVal
		}
	}

	switch strings.ToLower(layer) {
	case "routeslines", "routes_lines", "routes":
		initRouteShapes(&e)

		tolSq := getSimplificationToleranceSq(zoom)
		var res []LayerRouteLine
		for _, rs := range routeShapesList {
			if bbox != nil {
				latMargin := math.Max(0.015, (bbox.MaxLat-bbox.MinLat)*0.25)
				lonMargin := math.Max(0.015, (bbox.MaxLon-bbox.MinLon)*0.25)
				if rs.MaxLat < bbox.MinLat-latMargin || rs.MinLat > bbox.MaxLat+latMargin ||
					rs.MaxLon < bbox.MinLon-lonMargin || rs.MinLon > bbox.MaxLon+lonMargin {
					continue
				}
				clippedSegments := clipPolylineToBBox(rs.Points, *bbox)
				for _, seg := range clippedSegments {
					pts := seg
					if tolSq > 0 && len(pts) > 2 {
						pts = simplifyPoints(pts, tolSq)
					}
					if len(pts) >= 2 {
						res = append(res, LayerRouteLine{
							RouteID: rs.RouteID,
							Color:   rs.Color,
							Points:  pts,
						})
					}
				}
			} else {
				pts := rs.Points
				if tolSq > 0 && len(pts) > 2 {
					pts = simplifyPoints(pts, tolSq)
				}
				if len(pts) >= 2 {
					res = append(res, LayerRouteLine{
						RouteID: rs.RouteID,
						Color:   rs.Color,
						Points:  pts,
					})
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"layer": "routesLines",
			"total": len(res),
			"lines": res,
		})

	case "stops":
		type candidateStop struct {
			item     LayerStopItem
			priority int
		}

		sourceStops := e.Net.ParentsStations
		if zoom >= 14 {
			initDetailedStops(&e)
			if len(detailedStopsList) > 0 {
				sourceStops = detailedStopsList
			}
		}

		var candidates []candidateStop
		seenStops := make(map[string]bool)

		for _, ps := range sourceStops {
			if ps.Lat == 0 && ps.Lon == 0 {
				continue
			}

			cleanPID := utils.IdRemoveIDFMPrefix(ps.PID)
			stopUniqueKey := cleanPID + ":" + strconv.Itoa(int(ps.PType)) + ":" + ps.Name
			if seenStops[stopUniqueKey] {
				continue
			}

			if bbox != nil {
				if ps.Lat < bbox.MinLat || ps.Lat > bbox.MaxLat ||
					ps.Lon < bbox.MinLon || ps.Lon > bbox.MaxLon {
					continue
				}
			}

			prio := getStopPriority(ps)
			if zoom > 0 && !matchesZoomLevel(prio, zoom) {
				continue
			}

			seenStops[stopUniqueKey] = true
			rInfo := buildStopRoutes(ps, zoom)

			candidates = append(candidates, candidateStop{
				priority: prio,
				item: LayerStopItem{
					ID:     cleanPID,
					Name:   ps.Name,
					Type:   ps.PType,
					Lat:    ps.Lat,
					Lon:    ps.Lon,
					Routes: rInfo,
				},
			})
		}

		slices.SortFunc(candidates, func(a, b candidateStop) int {
			if a.priority != b.priority {
				return cmp.Compare(b.priority, a.priority)
			}
			return strings.Compare(a.item.Name, b.item.Name)
		})

		distThreshold := 0.05
		if zoom > 0 {
			distThreshold = getProximityThreshold(zoom)
		}

		var filtered []LayerStopItem
		for _, cand := range candidates {
			tooClose := false
			for i := range filtered {
				dist := utils.CoordsGetDistance(cand.item.Lon, cand.item.Lat, filtered[i].Lon, filtered[i].Lat)
				if dist < distThreshold {
					mergedSeenPictos := make(map[string]bool)
					mergedSeenNames := make(map[string]bool)
					for _, r := range filtered[i].Routes {
						if r.Picto != "" {
							mergedSeenPictos[r.Picto] = true
						} else if r.Name != "" {
							mergedSeenNames[strconv.Itoa(int(r.Type))+":"+r.Name] = true
						}
					}
					for _, r := range cand.item.Routes {
						if r.Picto != "" {
							if !mergedSeenPictos[r.Picto] {
								mergedSeenPictos[r.Picto] = true
								filtered[i].Routes = append(filtered[i].Routes, r)
							}
						} else if r.Name != "" {
							key := strconv.Itoa(int(r.Type)) + ":" + r.Name
							if !mergedSeenNames[key] {
								mergedSeenNames[key] = true
								filtered[i].Routes = append(filtered[i].Routes, r)
							}
						}
					}
					tooClose = true
					break
				}
			}
			if !tooClose {
				filtered = append(filtered, cand.item)
				if len(filtered) >= limit {
					break
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"layer": "stops",
			"total": len(filtered),
			"stops": filtered,
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unknown layer. Valid layers are 'routesLines' and 'stops'",
		})
	}
}

func init() {
	Register(ApiRoute{
		Method:  http.MethodGet,
		Path:    "/layers/:layer/:x/:y/:z",
		Handler: handleLayerRequest,
	})

	Register(ApiRoute{
		Method:  http.MethodGet,
		Path:    "/layers/:layer",
		Handler: handleLayerRequest,
	})

	Register(ApiRoute{
		Method:  http.MethodGet,
		Path:    "/layers",
		Handler: handleLayerRequest,
	})
}
