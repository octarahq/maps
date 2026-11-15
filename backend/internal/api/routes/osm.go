package routes

import (
	"net/http"
	"strconv"

	"maps/backend/internal/engine"
	"maps/backend/internal/osm"

	"github.com/gin-gonic/gin"
)

func init() {
	Register(ApiRoute{
		Method:  http.MethodGet,
		Path:    "/osm/platform",
		Handler: handleOSMPlatform,
	})
	Register(ApiRoute{
		Method:  http.MethodGet,
		Path:    "/osm/transfer-guide",
		Handler: handleOSMTransferGuide,
	})
}

func handleOSMTransferGuide(c *gin.Context, _ engine.Engine) {
	fromLat, errFLat := strconv.ParseFloat(c.Query("from_lat"), 64)
	fromLon, errFLon := strconv.ParseFloat(c.Query("from_lon"), 64)
	toLat, errTLat := strconv.ParseFloat(c.Query("to_lat"), 64)
	toLon, errTLon := strconv.ParseFloat(c.Query("to_lon"), 64)
	if errFLat != nil || errFLon != nil || errTLat != nil || errTLon != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid from_lat, from_lon, to_lat, to_lon are required"})
		return
	}

	fromLevel := c.Query("from_level")
	toLevel := c.Query("to_level")
	fromNearest := c.Query("from_nearest") == "true"
	toNearest := c.Query("to_nearest") == "true"
	fromRef := c.Query("from_ref")
	if fromRef != "" || fromNearest {
		if p, ok := osm.FindPlatformWithFallback(fromLat, fromLon, fromRef, fromNearest); ok {
			fromLat, fromLon = p.Lat, p.Lon
			if fromLevel == "" {
				fromLevel = p.Level
			}
		}
	}
	toRef := c.Query("to_ref")
	if toRef != "" || toNearest {
		if p, ok := osm.FindPlatformWithFallback(toLat, toLon, toRef, toNearest); ok {
			toLat, toLon = p.Lat, p.Lon
			if toLevel == "" {
				toLevel = p.Level
			}
		}
	}

	wheelchair := c.Query("wheelchair") == "true"
	res := osm.FindIndoorRoute(fromLat, fromLon, fromLevel, toLat, toLon, toLevel, wheelchair)
	c.JSON(http.StatusOK, res)
}

func handleOSMPlatform(c *gin.Context, _ engine.Engine) {
	lat, errLat := strconv.ParseFloat(c.Query("lat"), 64)
	lon, errLon := strconv.ParseFloat(c.Query("lon"), 64)
	if errLat != nil || errLon != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid lat and lon are required"})
		return
	}

	fromLat, errFromLat := strconv.ParseFloat(c.Query("from_lat"), 64)
	fromLon, errFromLon := strconv.ParseFloat(c.Query("from_lon"), 64)
	nextLat, errNextLat := strconv.ParseFloat(c.Query("next_lat"), 64)
	nextLon, errNextLon := strconv.ParseFloat(c.Query("next_lon"), 64)

	res := osm.ResolvePlatform(osm.PlatformQuery{
		Lat:          lat,
		Lon:          lon,
		Ref:          c.Query("ref"),
		AllowNearest: c.Query("allow_nearest") == "true",
		FromLat:      fromLat,
		FromLon:      fromLon,
		HasFrom:      errFromLat == nil && errFromLon == nil,
		NextLat:      nextLat,
		NextLon:      nextLon,
		HasNext:      errNextLat == nil && errNextLon == nil,
		Headsign:     c.Query("headsign"),
		Line:         c.Query("line"),
		Mode:         c.Query("mode"),
	})
	if !res.Found {
		c.JSON(http.StatusOK, gin.H{
			"found": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"found":     true,
		"ref":       res.Platform.Ref,
		"level":     res.Platform.Level,
		"lat":       res.Platform.Lat,
		"lon":       res.Platform.Lon,
		"exit_side": res.ExitSide,
	})
}
