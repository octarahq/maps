package utils

import "strings"

func IdRemoveIDFMPrefix(id string) string {
	var res string = id
	res = strings.ReplaceAll(id, "IDFM:", "")
	res = strings.ReplaceAll(id, "monomodalStopPlace:", "")
	return res
}
