package routes

import (
	"maps/backend/internal/engine"
	"maps/backend/internal/osm"
	"maps/backend/internal/types"
	"maps/backend/internal/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func parseCoordinate(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

func parseCoordinatesPair(s string) (float64, float64, bool) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	v1, err1 := parseCoordinate(parts[0])
	v2, err2 := parseCoordinate(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	if v1 >= 40 && v1 <= 55 && (v2 < 40 || v2 > 55) {
		return v2, v1, true
	}
	if v2 >= 40 && v2 <= 55 && (v1 < 40 || v1 > 55) {
		return v1, v2, true
	}
	return v2, v1, true
}

func resolvePoint(c *gin.Context, prefix string, e *engine.Engine) (float64, float64, bool) {
	latStr := c.Query(prefix + "_lat")
	if latStr == "" {
		latStr = c.Query("lat_" + prefix)
	}
	lonStr := c.Query(prefix + "_lon")
	if lonStr == "" {
		lonStr = c.Query(prefix + "_lng")
	}
	if lonStr == "" {
		lonStr = c.Query("lon_" + prefix)
	}
	if lonStr == "" {
		lonStr = c.Query("lng_" + prefix)
	}

	if latStr != "" && lonStr != "" {
		lat, err1 := parseCoordinate(latStr)
		lon, err2 := parseCoordinate(lonStr)
		if err1 == nil && err2 == nil {
			return lon, lat, true
		}
	}

	stopQuery := c.Query(prefix + "_stop")
	if stopQuery == "" {
		stopQuery = c.Query(prefix + "_sid")
	}
	if stopQuery != "" {
		if stop, ok := e.SearchStopById(stopQuery); ok && (stop.Lat != 0 || stop.Long != 0) {
			return stop.Long, stop.Lat, true
		}
	}

	generic := strings.TrimSpace(c.Query(prefix))
	if generic != "" {
		if strings.Contains(generic, ",") {
			if lon, lat, ok := parseCoordinatesPair(generic); ok {
				return lon, lat, true
			}
		}
		if stop, ok := e.SearchStopById(generic); ok && (stop.Lat != 0 || stop.Long != 0) {
			return stop.Long, stop.Lat, true
		}
	}

	return 0, 0, false
}

func init() {
	Register(ApiRoute{
		Method: http.MethodGet,
		Path:   "/transit/journey",
		Handler: func(c *gin.Context, e engine.Engine) {
			targetTime := utils.NowInParis().Truncate(time.Second)
			targetTimeType := types.TimeTypeDeparture

			if value := c.Query("arrivaltime"); value != "" {
				if parsedTime, ok := utils.ParseCustomTimeInParis(value); ok {
					targetTime = parsedTime
				}
				targetTimeType = types.TimeTypeArrival
			} else if value := c.Query("departuretime"); value != "" {
				if parsedTime, ok := utils.ParseCustomTimeInParis(value); ok {
					targetTime = parsedTime
				}
			} else if value := c.Query("time"); value != "" {
				if parsedTime, ok := utils.ParseCustomTimeInParis(value); ok {
					targetTime = parsedTime
				}
				if strings.ToLower(c.Query("timetype")) == "arrival" || strings.ToLower(c.Query("time_type")) == "arrival" {
					targetTimeType = types.TimeTypeArrival
				}
			}

			allowedModes := []types.RouteType{
				types.RouteTypeAerialLift,
				types.RouteTypeBus,
				types.RouteTypeCableTram,
				types.RouteTypeFerry,
				types.RouteTypeFunicular,
				types.RouteTypeMetro,
				types.RouteTypeTrain,
				types.RouteTypeTram,
			}

			if value := c.Query("allowedmodes"); value != "" {
				sModes := strings.Split(value, ",")
				var modes []types.RouteType
				for _, m := range sModes {
					switch strings.ToLower(strings.TrimSpace(m)) {
					case "aeriallift":
						modes = append(modes, types.RouteTypeAerialLift)
					case "bus":
						modes = append(modes, types.RouteTypeBus)
					case "cabletram":
						modes = append(modes, types.RouteTypeCableTram)
					case "ferry":
						modes = append(modes, types.RouteTypeFerry)
					case "funicular":
						modes = append(modes, types.RouteTypeFunicular)
					case "metro":
						modes = append(modes, types.RouteTypeMetro)
					case "train":
						modes = append(modes, types.RouteTypeTrain)
					case "tram":
						modes = append(modes, types.RouteTypeTram)
					}
				}
				if len(modes) > 0 {
					allowedModes = modes
				}
			}

			var weelchair types.Accessibility = types.AccessibilityUnknown
			if value := strings.ToLower(strings.TrimSpace(c.Query("wheelchair"))); value != "" {
				switch value {
				case "true":
					weelchair = types.AccessibilityYes
				case "false":
					weelchair = types.AccessibilityUnknown
				}
			}

			lonFrom, latFrom, okFrom := resolvePoint(c, "from", &e)
			if !okFrom {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "missing or invalid departure ('from', 'from_lat'/'from_lon', or 'from_stop')",
				})
				return
			}

			lonTo, latTo, okTo := resolvePoint(c, "to", &e)
			if !okTo {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "missing or invalid arrival ('to', 'to_lat'/'to_lon', or 'to_stop')",
				})
				return
			}

			config := types.CalculatorConfig{
				TargetTime:     targetTime,
				TimeType:       targetTimeType,
				AllowedModes:   allowedModes,
				WithBike:       c.Query("withbike") == "true",
				WheelchairOnly: weelchair,
			}

			edges, err := e.CalculateRouteCordsToCords(lonFrom, latFrom, lonTo, latTo, config)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			journey := make(map[string]*types.ParsedTrip)
			for n, route := range edges {
				if len(route) > 0 {
					journey[n] = e.EdgesToParsedTrip(route, lonFrom, latFrom, lonTo, latTo, config)
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"journey": journey,
			})
		},
	})

	Register(ApiRoute{
		Method:  http.MethodGet,
		Path:    "/transit/leg-live",
		Handler: handleTransitLegLive,
	})
}

func parseIsoOrParisTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, true
	}
	return utils.ParseCustomTimeInParis(raw)
}

func handleTransitLegLive(c *gin.Context, e engine.Engine) {
	rid := strings.TrimSpace(c.Query("rid"))
	fromSID := strings.TrimSpace(c.Query("from_sid"))
	toSID := strings.TrimSpace(c.Query("to_sid"))
	if rid == "" || (fromSID == "" && toSID == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rid and from_sid/to_sid are required"})
		return
	}

	var tripIndex uint32
	if rawIdx := strings.TrimSpace(c.Query("trip_index")); rawIdx != "" {
		if parsed, err := strconv.ParseUint(rawIdx, 10, 32); err == nil {
			tripIndex = uint32(parsed)
		}
	}

	schedDep, _ := parseIsoOrParisTime(c.Query("sched_dep"))
	if schedDep.IsZero() {
		schedDep, _ = parseIsoOrParisTime(c.Query("departure_time"))
	}
	schedArr, _ := parseIsoOrParisTime(c.Query("sched_arr"))
	if schedArr.IsZero() {
		schedArr, _ = parseIsoOrParisTime(c.Query("arrival_time"))
	}

	headsign := strings.TrimSpace(c.Query("headsign"))

	res := e.ResolveLegLive(engine.LegLiveQuery{
		RID:       rid,
		FromSID:   fromSID,
		ToSID:     toSID,
		TripIndex: tripIndex,
		SchedDep:  schedDep,
		SchedArr:  schedArr,
		Headsign:  headsign,
	})

	mode := strings.TrimSpace(c.Query("mode"))
	line := strings.TrimSpace(c.Query("line"))
	isTrain := e.IsTrainRoute(rid)
	allowNearest := !isTrain
	if mode == "" || line == "" {

		cleanRID := utils.IdRemoveIDFMPrefix(rid)
		for _, r := range e.Net.Routes {
			if r.RID == rid || utils.IdRemoveIDFMPrefix(r.RID) == cleanRID {
				if line == "" {
					line = r.SName
				}
				if mode == "" {
					switch r.RType {
					case types.RouteTypeMetro:
						mode = "subway"
					case types.RouteTypeTram:
						mode = "tram"
					case types.RouteTypeTrain:
						mode = "train"
					}
				}
				break
			}
		}
	}
	if mode == "subway" || mode == "tram" || mode == "light_rail" {
		allowNearest = true
	}

	var fromLat, fromLon, toLat, toLon float64
	hasFromCoords, hasToCoords := false, false
	if vLat, err1 := parseCoordinate(c.Query("from_lat")); err1 == nil {
		if vLon, err2 := parseCoordinate(c.Query("from_lon")); err2 == nil {
			fromLat, fromLon, hasFromCoords = vLat, vLon, true
		}
	}
	if !hasFromCoords && fromSID != "" {
		if s, ok := e.SearchStopById(fromSID); ok && (s.Lat != 0 || s.Long != 0) {
			fromLat, fromLon, hasFromCoords = s.Lat, s.Long, true
		}
	}
	if vLat, err1 := parseCoordinate(c.Query("to_lat")); err1 == nil {
		if vLon, err2 := parseCoordinate(c.Query("to_lon")); err2 == nil {
			toLat, toLon, hasToCoords = vLat, vLon, true
		}
	}
	if !hasToCoords && toSID != "" {
		if s, ok := e.SearchStopById(toSID); ok && (s.Lat != 0 || s.Long != 0) {
			toLat, toLon, hasToCoords = s.Lat, s.Long, true
		}
	}

	prevLat, prevLon, hasPrev := fromLat, fromLon, hasFromCoords
	if vLat, err1 := parseCoordinate(c.Query("prev_lat")); err1 == nil {
		if vLon, err2 := parseCoordinate(c.Query("prev_lon")); err2 == nil {
			prevLat, prevLon, hasPrev = vLat, vLon, true
		}
	}

	arrExitSide, arrOsmRef, arrLevel := "", "", ""
	if hasToCoords && (res.ArrivalPlatform != "" || allowNearest) {
		pRes := osm.ResolvePlatform(osm.PlatformQuery{
			Lat:          toLat,
			Lon:          toLon,
			Ref:          res.ArrivalPlatform,
			AllowNearest: allowNearest,
			FromLat:      prevLat,
			FromLon:      prevLon,
			HasFrom:      hasPrev,
			Headsign:     headsign,
			Line:         line,
			Mode:         mode,
		})
		if pRes.Found {
			arrExitSide = pRes.ExitSide
			arrOsmRef = pRes.Platform.Ref
			arrLevel = pRes.Platform.Level
		}
	}

	depExitSide, depOsmRef, depLevel := "", "", ""
	if hasFromCoords && (res.DeparturePlatform != "" || allowNearest) {
		pRes := osm.ResolvePlatform(osm.PlatformQuery{
			Lat:          fromLat,
			Lon:          fromLon,
			Ref:          res.DeparturePlatform,
			AllowNearest: allowNearest,
			NextLat:      toLat,
			NextLon:      toLon,
			HasNext:      hasToCoords,
			Headsign:     headsign,
			Line:         line,
			Mode:         mode,
		})
		if pRes.Found {
			depExitSide = pRes.ExitSide
			depOsmRef = pRes.Platform.Ref
			depLevel = pRes.Platform.Level
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"found":               res.Found,
		"cancelled":           res.Cancelled,
		"dep_delay_sec":       res.DepDelaySec,
		"arr_delay_sec":       res.ArrDelaySec,
		"real_dep":            res.RealDep,
		"real_arr":            res.RealArr,
		"departure_platform":  res.DeparturePlatform,
		"arrival_platform":    res.ArrivalPlatform,
		"is_long_train":       res.IsLongTrain,
		"source":              res.Source,
		"arrival_exit_side":   arrExitSide,
		"arrival_osm_ref":     arrOsmRef,
		"arrival_level":       arrLevel,
		"departure_exit_side": depExitSide,
		"departure_osm_ref":   depOsmRef,
		"departure_level":     depLevel,
	})
}
