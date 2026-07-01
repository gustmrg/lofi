package version

import "testing"

func TestVersionDisplay(t *testing.T) {
	if got := Display("v1.2.3"); got != "LoFi v1.2.3" {
		t.Fatalf("Display() = %q", got)
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "equal with v prefix", a: "v1.2.3", b: "1.2.3", want: 0},
		{name: "newer minor", a: "1.3.0", b: "1.2.9", want: 1},
		{name: "older major", a: "1.9.9", b: "2.0.0", want: -1},
		{name: "release after prerelease", a: "1.0.0", b: "1.0.0-rc.1", want: 1},
		{name: "prerelease order", a: "1.0.0-rc.2", b: "1.0.0-rc.1", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compare(tt.a, tt.b)
			if err != nil {
				t.Fatalf("Compare(): %v", err)
			}
			if got != tt.want {
				t.Fatalf("Compare() = %d, want %d", got, tt.want)
			}
		})
	}
}
