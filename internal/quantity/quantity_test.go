package quantity

import "testing"

func TestParseBytes(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"1Gi", 1 << 30},
		{"512Mi", 512 << 20},
		{"1.5Gi", 1536 << 20},
		{"128974848", 128974848},
		{"129M", 129e6},
		{"1G", 1e9},
		{"100k", 100e3},
		{"64Ki", 64 << 10},
		{"1e3", 1000},
	}
	for _, tt := range tests {
		got, err := ParseBytes(tt.in)
		if err != nil {
			t.Errorf("ParseBytes(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseBytes(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestParseBytesInvalid(t *testing.T) {
	for _, in := range []string{"", "abc", "1Zi", "-1Gi", "Gi"} {
		if _, err := ParseBytes(in); err == nil {
			t.Errorf("ParseBytes(%q) want error", in)
		}
	}
}

func TestParseMilliCPU(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"500m", 500},
		{"1", 1000},
		{"1.5", 1500},
		{"0.1", 100},
		{"2000m", 2000},
		{"250m", 250},
	}
	for _, tt := range tests {
		got, err := ParseMilliCPU(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("ParseMilliCPU(%q) = %d, %v; want %d", tt.in, got, err, tt.want)
		}
	}
}

func TestFormat(t *testing.T) {
	if got := Human(1 << 30); got != "1Gi" {
		t.Errorf("Human(1Gi) = %s", got)
	}
	if got := Human(1280 << 20); got != "1.25Gi" {
		t.Errorf("Human(1.25Gi) = %s", got)
	}
	if got := Human(900 << 20); got != "900Mi" {
		t.Errorf("Human(900Mi) = %s", got)
	}
	for milli, want := range map[int64]string{500: "500m", 1000: "1", 1500: "1500m", 250: "250m"} {
		if got := CPU(milli); got != want {
			t.Errorf("CPU(%d) = %s, want %s", milli, got, want)
		}
	}
}
