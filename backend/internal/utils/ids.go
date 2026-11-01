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
