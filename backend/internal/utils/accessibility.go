package utils

import "maps/backend/internal/types"

func IsAccessible(access types.Accessibility, required types.Accessibility) bool {
	switch required {
	case types.AccessibilityNo:
		return true
	case types.AccessibilityUnknown:
		return access == types.AccessibilityUnknown || access == types.AccessibilityYes
	case types.AccessibilityYes:
		return access == types.AccessibilityYes
	default:
		return true
	}
}
