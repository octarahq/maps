package utils

import "strings"

func IdRemoveIDFMPrefix(id string) string {
	var res string = id
	res = strings.ReplaceAll(res, "stop_area:IDFM:", "")
	res = strings.ReplaceAll(res, "stop_point:IDFM:", "")
	res = strings.ReplaceAll(res, "line:IDFM:", "")
	res = strings.ReplaceAll(res, "stop_area:", "")
	res = strings.ReplaceAll(res, "stop_point:", "")
	res = strings.ReplaceAll(res, "line:", "")
	res = strings.ReplaceAll(res, "IDFM:", "")
	res = strings.ReplaceAll(res, "monomodalStopPlace:", "")
	res = strings.ReplaceAll(res, "StopPlaceEntrance:", "")
	res = strings.ReplaceAll(res, "StopPlace:", "")
	return strings.TrimSpace(res)
}

func ExtractPictoID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	const marker = "/files/"
	if idx := strings.Index(raw, marker); idx != -1 {
		rest := raw[idx+len(marker):]
		end := strings.IndexAny(rest, "/?# )\"'\n\r\t")
		if end != -1 {
			return rest[:end]
		}
		return rest
	}

	raw = strings.Trim(raw, "()[]")
	if end := strings.IndexAny(raw, "/?# )\"'\n\r\t"); end != -1 {
		return raw[:end]
	}
	return raw
}
