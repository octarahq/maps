package routes

import (
	"maps/backend/internal/engine"
	"maps/backend/internal/types"
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
			location, _ := time.LoadLocation("Europe/Paris")
			targetTime := time.Now().In(location).Truncate(time.Second)
			targetTimeType := types.TimeTypeDeparture

			parseCustomTime := func(val string) (time.Time, bool) {
				if parsed, err := time.Parse(time.RFC3339, val); err == nil {
					return parsed.In(location).Truncate(time.Second), true
				}
				if parsed, err := time.ParseInLocation("2006-01-02T15:04:05", val, location); err == nil {
					return parsed.Truncate(time.Second), true
				}
				if parsed, err := time.ParseInLocation("2006-01-02T15:04", val, location); err == nil {
					return parsed.Truncate(time.Second), true
				}
				if parsed, err := time.ParseInLocation("15:04", val, location); err == nil {
					now := time.Now().In(location)
					return time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, location), true
				}
				return time.Time{}, false
			}


			if value := c.Query("arrivaltime"); value != "" {
				if parsedTime, ok := parseCustomTime(value); ok {
					targetTime = parsedTime
				}
				targetTimeType = types.TimeTypeArrival
			} else if value := c.Query("departuretime"); value != "" {
				if parsedTime, ok := parseCustomTime(value); ok {
					targetTime = parsedTime
				}
			} else if value := c.Query("time"); value != "" {
				if parsedTime, ok := parseCustomTime(value); ok {
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
				case "obligatory", "yes", "true", "1":
					weelchair = types.AccessibilityYes
				case "no", "false", "0":
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
}
