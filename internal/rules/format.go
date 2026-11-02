package rules

import "strconv"

// pct formats a percentage without trailing zeros: 75, 62.5.
func pct(p float64) string {
	return strconv.FormatFloat(p, 'f', -1, 64) + "%"
}
