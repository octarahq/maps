package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"maps/backend/internal/engine"
	"maps/backend/internal/env"
	"maps/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type AddressResult struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Name        string   `json:"name"`
	HouseNumber string   `json:"house_number,omitempty"`
	Street      string   `json:"street,omitempty"`
	Postcode    string   `json:"postcode,omitempty"`
	City        string   `json:"city,omitempty"`
	Country     string   `json:"country,omitempty"`
	Type        string   `json:"type"`
	Lat         float64  `json:"lat"`
	Lon         float64  `json:"lon"`
	DistanceM   *float64 `json:"distance_m,omitempty"`
	Score       float64  `json:"score"`
}

type nominatimResponse struct {
	PlaceID     int64            `json:"place_id"`
	OSMType     string           `json:"osm_type"`
	OSMID       int64            `json:"osm_id"`
	Lat         string           `json:"lat"`
	Lon         string           `json:"lon"`
	Category    string           `json:"category"`
	Type        string           `json:"type"`
	Importance  float64          `json:"importance"`
	DisplayName string           `json:"display_name"`
	Name        string           `json:"name"`
	Address     nominatimAddress `json:"address"`
}

type nominatimAddress struct {
	HouseNumber  string `json:"house_number"`
	Road         string `json:"road"`
	Pedestrian   string `json:"pedestrian"`
	Footway      string `json:"footway"`
	Cycleway     string `json:"cycleway"`
	Suburb       string `json:"suburb"`
	CityDistrict string `json:"city_district"`
	City         string `json:"city"`
	Town         string `json:"town"`
	Village      string `json:"village"`
	Municipality string `json:"municipality"`
	Postcode     string `json:"postcode"`
	Country      string `json:"country"`
}

var nominatimHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
}

func init() {
	Register(ApiRoute{
		Method:  http.MethodGet,
		Path:    "/addresses/search",
		Handler: handleAddressSearch,
	})
}

func handleAddressSearch(c *gin.Context, _ engine.Engine) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) <= 2 {
		c.JSON(http.StatusOK, gin.H{
			"total":     0,
			"addresses": []AddressResult{},
		})
		return
	}

	limit := 10
	if sLimit := c.Query("limit"); sLimit != "" {
		if parsed, err := strconv.Atoi(sLimit); err == nil && parsed > 0 {
			limit = min(parsed, 50)
		}
	}

	sLat, sLon := c.Query("lat"), c.Query("lon")
	hasCoords := false
	var lat, lon float64
	if sLat != "" || sLon != "" {
		if sLat == "" || sLon == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "need to have both coordinates",
			})
			return
		}
		var errLat, errLon error
		lat, errLat = strconv.ParseFloat(sLat, 64)
		lon, errLon = strconv.ParseFloat(sLon, 64)
		if errLat != nil || errLon != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid lat or lon coordinate",
			})
			return
		}
		hasCoords = true
	}

	countryCodes := strings.TrimSpace(c.Query("countrycodes"))
	if countryCodes == "" {
		countryCodes = "fr"
	}
	typeFilter := strings.ToLower(strings.TrimSpace(c.Query("type")))

	fetchLimit := max(limit*3, 25)
	rawResults, err := queryNominatim(q, countryCodes, fetchLimit)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "failed to query address provider",
		})
		return
	}

	results := filterAndScoreAddresses(rawResults, q, typeFilter, hasCoords, lat, lon)
	if len(results) > limit {
		results = results[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     len(results),
		"addresses": results,
	})
}

func queryNominatim(q, countryCodes string, limit int) ([]nominatimResponse, error) {
	baseURL := strings.TrimRight(env.GetEnv("NOMINATIM_URL"), "/")
	if baseURL == "" {
		baseURL = "https://nominatim.openstreetmap.org"
	}

	params := url.Values{}
	params.Set("q", q)
	params.Set("format", "jsonv2")
	params.Set("addressdetails", "1")
	params.Set("dedupe", "1")
	params.Set("limit", strconv.Itoa(limit))
	if countryCodes != "" && countryCodes != "all" {
		params.Set("countrycodes", countryCodes)
	}

	reqURL := fmt.Sprintf("%s/search?%s", baseURL, params.Encode())
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OctaraMaps/1.0 (local-api)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "fr,en;q=0.8")

	resp, err := nominatimHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nominatim returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []nominatimResponse
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func filterAndScoreAddresses(raw []nominatimResponse, query, typeFilter string, hasCoords bool, refLat, refLon float64) []AddressResult {
	normQuery := normalizeText(query)
	queryTokens := strings.Fields(normQuery)

	seen := make(map[string]struct{}, len(raw))
	results := make([]AddressResult, 0, len(raw))

	for _, item := range raw {
		lat, errLat := strconv.ParseFloat(item.Lat, 64)
		lon, errLon := strconv.ParseFloat(item.Lon, 64)
		if errLat != nil || errLon != nil || (lat == 0 && lon == 0) {
			continue
		}

		addrType := classifyAddressType(item)
		if typeFilter != "" && addrType != typeFilter {
			continue
		}

		street := firstNonEmpty(item.Address.Road, item.Address.Pedestrian, item.Address.Footway, item.Address.Cycleway)
		city := firstNonEmpty(item.Address.City, item.Address.Town, item.Address.Village, item.Address.Municipality)
		name := buildShortName(item.Name, item.Address.HouseNumber, street, city, item.DisplayName)
		label := strings.TrimSpace(item.DisplayName)
		if label == "" {
			label = name
		}

		dedupKey := fmt.Sprintf("%s|%s|%s", normalizeText(name), normalizeText(item.Address.Postcode), normalizeText(city))
		if _, exists := seen[dedupKey]; exists {
			continue
		}
		seen[dedupKey] = struct{}{}

		var distPtr *float64
		if hasCoords {
			d := utils.CoordsGetDistance(lon, lat, refLon, refLat) * 1000.0
			distPtr = &d
		}

		score := computeAddressScore(normQuery, queryTokens, name, label, addrType, item.Importance, distPtr)

		results = append(results, AddressResult{
			ID:          fmt.Sprintf("%s:%d", item.OSMType, item.OSMID),
			Label:       label,
			Name:        name,
			HouseNumber: item.Address.HouseNumber,
			Street:      street,
			Postcode:    item.Address.Postcode,
			City:        city,
			Country:     item.Address.Country,
			Type:        addrType,
			Lat:         lat,
			Lon:         lon,
			DistanceM:   distPtr,
			Score:       math.Round(score*1000) / 1000,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].DistanceM != nil && results[j].DistanceM != nil {
			return *results[i].DistanceM < *results[j].DistanceM
		}
		return results[i].Label < results[j].Label
	})

	return results
}

func classifyAddressType(item nominatimResponse) string {
	if item.Address.HouseNumber != "" {
		return "house"
	}
	switch item.Category {
	case "highway":
		return "street"
	case "place":
		switch item.Type {
		case "city", "town", "village", "municipality", "hamlet", "suburb":
			return "city"
		}
	case "boundary":
		if item.Type == "administrative" {
			return "city"
		}
	}
	if item.Address.Road != "" && item.Name == "" {
		return "street"
	}
	return "poi"
}

func buildShortName(name, houseNumber, street, city, displayName string) string {
	if houseNumber != "" && street != "" {
		return houseNumber + " " + street
	}
	if name != "" {
		return name
	}
	if street != "" {
		return street
	}
	if city != "" {
		return city
	}
	parts := strings.Split(displayName, ",")
	return strings.TrimSpace(parts[0])
}

func computeAddressScore(normQuery string, queryTokens []string, name, label, addrType string, importance float64, distM *float64) float64 {
	normName := normalizeText(name)
	normLabel := normalizeText(label)

	var textScore float64
	switch {
	case normName == normQuery:
		textScore = 1.0
	case strings.HasPrefix(normName, normQuery):
		textScore = 0.85
	case strings.Contains(normName, normQuery):
		textScore = 0.70
	default:
		matched := 0
		for _, tok := range queryTokens {
			if strings.Contains(normLabel, tok) {
				matched++
			}
		}
		if len(queryTokens) > 0 {
			textScore = 0.55 * (float64(matched) / float64(len(queryTokens)))
		}
	}

	typeWeight := map[string]float64{
		"house":  0.20,
		"street": 0.15,
		"poi":    0.12,
		"city":   0.10,
	}[addrType]

	impScore := math.Min(math.Max(importance, 0), 1.0) * 0.15

	score := textScore*0.55 + typeWeight + impScore
	if distM != nil {

		prox := 0.25 / (1.0 + (*distM / 5000.0))
		score += prox
	}
	return score
}

func normalizeText(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	res, _, err := transform.String(t, strings.ToLower(strings.TrimSpace(s)))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(s))
	}
	return res
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
