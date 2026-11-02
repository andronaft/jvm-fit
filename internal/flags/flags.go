// Package flags parses JVM command-line options that matter for memory and
// CPU: heap size, RAM percentages, thread stacks, container support and GC.
package flags

import (
	"fmt"
	"strconv"
	"strings"
)

// Source says where a group of options came from.
type Source struct {
	// Name is an env var (JAVA_TOOL_OPTIONS) or "command"/"args".
	Name string
	// Assumed is true when the JVM reads the options only if the image
	// entrypoint passes them on (JAVA_OPTS and friends).
	Assumed bool
}

// Value is one parsed option with the source that set it last.
type Value[T any] struct {
	V      T
	Raw    string
	Source Source
	Set    bool
}

func (v *Value[T]) set(val T, raw string, src Source) {
	*v = Value[T]{V: val, Raw: raw, Source: src, Set: true}
}

// Flags is what the JVM ends up with after all sources are applied in order.
// Later options win, like in the JVM itself.
type Flags struct {
	MaxHeap              Value[int64] // -Xmx, -XX:MaxHeapSize
	InitialHeap          Value[int64] // -Xms, -XX:InitialHeapSize
	ThreadStack          Value[int64] // -Xss, -XX:ThreadStackSize (KiB)
	MaxRAMPercentage     Value[float64]
	InitialRAMPercentage Value[float64]
	MinRAMPercentage     Value[float64]
	MaxRAM               Value[int64]
	MaxMetaspace         Value[int64]
	ReservedCodeCache    Value[int64]
	MaxDirectMemory      Value[int64]
	ActiveProcessorCount Value[int]
	UseContainerSupport  Value[bool]
	// GC is "Serial", "Parallel", "G1", "Z", "Shenandoah" or an OpenJ9 policy.
	GC Value[string]

	// Seen lists every recognized option in order, for reports.
	Seen []Option
}

// Option is one recognized option as written by the user.
type Option struct {
	Raw    string
	Source Source
}

// Warning is an option that looks like a memory or CPU option but could not be parsed.
type Warning struct {
	Raw    string
	Source Source
	Err    error
}

func (w Warning) String() string {
	return fmt.Sprintf("%s (%s): %v", w.Raw, w.Source.Name, w.Err)
}

var gcFlags = map[string]string{
	"UseSerialGC":        "Serial",
	"UseParallelGC":      "Parallel",
	"UseParallelOldGC":   "Parallel",
	"UseG1GC":            "G1",
	"UseZGC":             "Z",
	"UseShenandoahGC":    "Shenandoah",
	"UseConcMarkSweepGC": "CMS",
	"UseEpsilonGC":       "Epsilon",
}

// Apply parses tokens from one source and applies them on top of f.
// Tokens that are not JVM options (app arguments, -jar, class names) are skipped.
func (f *Flags) Apply(tokens []string, src Source) []Warning {
	var warns []Warning
	for _, tok := range tokens {
		recognized, err := f.applyOne(tok, src)
		if err != nil {
			warns = append(warns, Warning{Raw: tok, Source: src, Err: err})
			continue
		}
		if recognized {
			f.Seen = append(f.Seen, Option{Raw: tok, Source: src})
		}
	}
	return warns
}

func (f *Flags) applyOne(tok string, src Source) (bool, error) {
	switch {
	case strings.HasPrefix(tok, "-Xmx"):
		return setSize(&f.MaxHeap, tok, strings.TrimPrefix(tok, "-Xmx"), src)
	case strings.HasPrefix(tok, "-Xms"):
		return setSize(&f.InitialHeap, tok, strings.TrimPrefix(tok, "-Xms"), src)
	case strings.HasPrefix(tok, "-Xss"):
		return setSize(&f.ThreadStack, tok, strings.TrimPrefix(tok, "-Xss"), src)
	case strings.HasPrefix(tok, "-Xgcpolicy:"):
		// OpenJ9 GC policy, e.g. -Xgcpolicy:gencon.
		f.GC.set(strings.TrimPrefix(tok, "-Xgcpolicy:"), tok, src)
		return true, nil
	case strings.HasPrefix(tok, "-XX:+"), strings.HasPrefix(tok, "-XX:-"):
		on := tok[4] == '+'
		name := tok[5:]
		if gc, ok := gcFlags[name]; ok {
			if on {
				f.GC.set(gc, tok, src)
			} else if f.GC.Set && f.GC.V == gc {
				f.GC = Value[string]{}
			}
			return true, nil
		}
		if name == "UseContainerSupport" {
			f.UseContainerSupport.set(on, tok, src)
			return true, nil
		}
		return false, nil
	case strings.HasPrefix(tok, "-XX:"):
		name, val, ok := strings.Cut(strings.TrimPrefix(tok, "-XX:"), "=")
		if !ok {
			return false, nil
		}
		return f.applyXX(name, val, tok, src)
	}
	return false, nil
}

func (f *Flags) applyXX(name, val, raw string, src Source) (bool, error) {
	switch name {
	case "MaxHeapSize":
		return setSize(&f.MaxHeap, raw, val, src)
	case "InitialHeapSize":
		return setSize(&f.InitialHeap, raw, val, src)
	case "ThreadStackSize":
		// ThreadStackSize is in KiB when it has no unit.
		n, err := ParseSize(val)
		if err != nil {
			return false, err
		}
		if isDigits(val) {
			n *= 1024
		}
		f.ThreadStack.set(n, raw, src)
		return true, nil
	case "MaxRAMPercentage":
		return setPercent(&f.MaxRAMPercentage, raw, val, src)
	case "InitialRAMPercentage":
		return setPercent(&f.InitialRAMPercentage, raw, val, src)
	case "MinRAMPercentage":
		return setPercent(&f.MinRAMPercentage, raw, val, src)
	case "MaxRAMFraction":
		// Old flag (JDK 8u131+): heap = RAM / fraction.
		n, err := strconv.Atoi(val)
		if err != nil || n <= 0 {
			return false, fmt.Errorf("invalid MaxRAMFraction %q", val)
		}
		f.MaxRAMPercentage.set(100/float64(n), raw, src)
		return true, nil
	case "MaxRAM":
		return setSize(&f.MaxRAM, raw, val, src)
	case "MaxMetaspaceSize":
		return setSize(&f.MaxMetaspace, raw, val, src)
	case "ReservedCodeCacheSize":
		return setSize(&f.ReservedCodeCache, raw, val, src)
	case "MaxDirectMemorySize":
		return setSize(&f.MaxDirectMemory, raw, val, src)
	case "ActiveProcessorCount":
		n, err := strconv.Atoi(val)
		if err != nil || n < -1 || n == 0 {
			return false, fmt.Errorf("invalid ActiveProcessorCount %q", val)
		}
		if n == -1 {
			f.ActiveProcessorCount = Value[int]{}
			return true, nil
		}
		f.ActiveProcessorCount.set(n, raw, src)
		return true, nil
	}
	return false, nil
}

func setSize(v *Value[int64], raw, val string, src Source) (bool, error) {
	n, err := ParseSize(val)
	if err != nil {
		return false, err
	}
	v.set(n, raw, src)
	return true, nil
}

func setPercent(v *Value[float64], raw, val string, src Source) (bool, error) {
	p, err := strconv.ParseFloat(val, 64)
	if err != nil || p < 0 || p > 100 {
		return false, fmt.Errorf("invalid percentage %q", val)
	}
	v.set(p, raw, src)
	return true, nil
}

// ParseSize parses a JVM size: a number with an optional k/K, m/M, g/G or t/T
// suffix. Without a suffix the number is bytes.
func ParseSize(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("missing size")
	}
	mult := int64(1)
	switch s[len(s)-1] {
	case 'k', 'K':
		mult = 1 << 10
	case 'm', 'M':
		mult = 1 << 20
	case 'g', 'G':
		mult = 1 << 30
	case 't', 'T':
		mult = 1 << 40
	}
	num := s
	if mult != 1 {
		num = s[:len(s)-1]
	}
	if !isDigits(num) {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	n, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	if n > (1<<62)/mult {
		return 0, fmt.Errorf("size %q is too large", s)
	}
	return n * mult, nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
