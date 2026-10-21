package utils

func LevenshteinDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	previous := make([]int, len(br)+1)
	for j := range previous {
		previous[j] = j
	}

	for i, left := range ar {
		current := make([]int, len(br)+1)
		current[0] = i + 1
		for j, right := range br {
			cost := 0
			if left != right {
				cost = 1
			}
			current[j+1] = minimum(current[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous = current
	}
	return previous[len(br)]
}

func SearchByLevenshtein(candidates []string, query string, maxDistance int) []string {
	if maxDistance < 0 {
		return nil
	}
	type match struct {
		value, distance string
	}
	matches := make([]struct {
		value    string
		distance int
		order    int
	}, 0)
	for i, candidate := range candidates {
		distance := LevenshteinDistance(candidate, query)
		if distance <= maxDistance {
			matches = append(matches, struct {
				value    string
				distance int
				order    int
			}{candidate, distance, i})
		}
	}
	for i := 1; i < len(matches); i++ {
		for j := i; j > 0 && matches[j].distance < matches[j-1].distance; j-- {
			matches[j], matches[j-1] = matches[j-1], matches[j]
		}
	}
	result := make([]string, len(matches))
	for i, match := range matches {
		result[i] = match.value
	}
	return result
}

func minimum(values ...int) int {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}
