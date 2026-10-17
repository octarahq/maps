package engine

import (
	"sort"
	"strings"

	"maps/api/internal/types"
	"maps/api/internal/utils"
)

func normalize(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"à", "a", "â", "a", "ä", "a",
		"î", "i", "ï", "i",
		"ô", "o", "ö", "o",
		"ù", "u", "û", "u", "ü", "u",
		"ç", "c",
		"-", " ",
		"'", " ",
	)
	return replacer.Replace(s)
}

func (e *Engine) SearchStopsByName(query string) []types.Stop {
	normQ := strings.TrimSpace(normalize(query))
	if normQ == "" {
		return nil
	}

	type candidate struct {
		stop  types.Stop
		score int
	}

	var candidates []candidate
	seenNames := make(map[string]bool)

	for _, stop := range e.Net.Stops {
		normName := normalize(stop.Name)

		if normName == normQ {
			candidates = append(candidates, candidate{stop: stop, score: 0})
			continue
		}

		if strings.HasPrefix(normName, normQ) {
			candidates = append(candidates, candidate{stop: stop, score: 10})
			continue
		}

		if strings.Contains(normName, normQ) {
			candidates = append(candidates, candidate{stop: stop, score: 20})
			continue
		}

		words := strings.Fields(normName)
		bestWordDist := 999
		for _, w := range words {
			if strings.HasPrefix(w, normQ) {
				bestWordDist = 15
				break
			}
			diff := len(w) - len(normQ)
			if diff < 0 {
				diff = -diff
			}
			if diff <= 2 {
				d := utils.LevenshteinDistance(w, normQ)
				if d < bestWordDist {
					bestWordDist = d
				}
			}
		}

		if bestWordDist <= 2 {
			candidates = append(candidates, candidate{stop: stop, score: 100 + bestWordDist})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score < candidates[j].score
		}
		return len(candidates[i].stop.Name) < len(candidates[j].stop.Name)
	})

	var result []types.Stop
	for _, c := range candidates {
		if !seenNames[c.stop.Name] {
			seenNames[c.stop.Name] = true
			result = append(result, c.stop)
		}
	}

	return result
}

func (e *Engine) SearchStopById(query string) (types.Stop, bool) {
	stops := utils.IndexStopsByID(e.Net.Stops)

	for id, stop := range stops {
		if id == query {
			return stop, true
		}
	}

	return types.Stop{}, false
}
