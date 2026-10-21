package utils

import "maps/backend/internal/types"

func CheckAllowedRouteType(rType types.RouteType, allowedTypes []types.RouteType) bool {
	for _, aType := range allowedTypes {
		if aType == rType {
			return true
		}
	}
	return false
}
