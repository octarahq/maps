package utils

import "strings"

func IdRemoveIDFMPrefix(id string) string {
	var res string = id
	res = strings.ReplaceAll(res, "IDFM:", "")
	res = strings.ReplaceAll(res, "monomodalStopPlace:", "")
	res = strings.ReplaceAll(res, "StopPlaceEntrance:", "")
	res = strings.ReplaceAll(res, "StopPlace:", "")
	return strings.TrimSpace(res)
}
