// Package quantity parses Kubernetes resource quantities ("1Gi", "500m")
// and formats byte counts for reports.
package quantity

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	KiB int64 = 1 << 10
	MiB int64 = 1 << 20
	GiB int64 = 1 << 30
)

var binarySuffixes = []struct {
	suffix string
	mult   float64
}{
	{"Ki", 1 << 10}, {"Mi", 1 << 20}, {"Gi", 1 << 30}, {"Ti", 1 << 40}, {"Pi", 1 << 50}, {"Ei", 1 << 60},
}

var decimalSuffixes = []struct {
	suffix string
	mult   float64
}{
	{"m", 1e-3}, {"k", 1e3}, {"M", 1e6}, {"G", 1e9}, {"T", 1e12}, {"P", 1e15}, {"E", 1e18},
}

// parse returns the value of a Kubernetes quantity as a float.
func parse(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty quantity")
	}
	num, mult := s, 1.0
	matched := false
	for _, b := range binarySuffixes {
		if strings.HasSuffix(s, b.suffix) {
			num, mult, matched = strings.TrimSuffix(s, b.suffix), b.mult, true
			break
		}
	}
	if !matched {
		for _, d := range decimalSuffixes {
			if strings.HasSuffix(s, d.suffix) {
				num, mult = strings.TrimSuffix(s, d.suffix), d.mult
				break
			}
		}
	}
	// Exponent form: 1e3, 1E3.
	v, err := strconv.ParseFloat(num, 64)
	if err != nil || v < 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, fmt.Errorf("invalid quantity %q", s)
	}
	return v * mult, nil
}

// ParseBytes parses a memory quantity such as "512Mi", "1G" or "1073741824".
func ParseBytes(s string) (int64, error) {
	v, err := parse(s)
	if err != nil {
		return 0, err
	}
	return int64(math.Ceil(v)), nil
}

// ParseMilliCPU parses a CPU quantity such as "500m", "1" or "1.5" into millicores.
func ParseMilliCPU(s string) (int64, error) {
	v, err := parse(s)
	if err != nil {
		return 0, err
	}
	return int64(math.Ceil(v * 1000)), nil
}

// Mi formats bytes as whole mebibytes, rounding to the nearest one.
func Mi(b int64) string {
	return fmt.Sprintf("%dMi", int64(math.Round(float64(b)/float64(MiB))))
}

// Human formats bytes with the largest binary unit that keeps the number short:
// 1Gi, 1.25Gi, 768Mi.
func Human(b int64) string {
	if b >= GiB && b%(GiB/4) == 0 {
		return strconv.FormatFloat(float64(b)/float64(GiB), 'f', -1, 64) + "Gi"
	}
	return Mi(b)
}

// CPU formats millicores the way people write them in manifests: 500m, 1, 1500m.
func CPU(milli int64) string {
	if milli%1000 == 0 {
		return strconv.FormatInt(milli/1000, 10)
	}
	return fmt.Sprintf("%dm", milli)
}
