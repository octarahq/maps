package utils

import "maps/backend/internal/types"

func IsAccessible(access types.Accessibility) bool {
	switch access {
	case types.AccessibilityUnknown:
	case types.AccessibilityYes:
		return true
	}

	return false
}
