package ui

import (
	"strings"
	"testing"
	"time"
)

func TestHumanDur(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Second, "45s"},
		{5 * time.Minute, "5m"},
		{3*time.Hour + 12*time.Minute, "3h 12m"},
		{2*24*time.Hour + 4*time.Hour, "2d 4h"},
	}
	for _, c := range cases {
		if got := HumanDur(c.d); got != c.want {
			t.Errorf("HumanDur(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestFmtExpiry(t *testing.T) {
	now := time.UnixMilli(1_000_000_000_000)
	in3h := now.Add(3 * time.Hour).UnixMilli()
	if got := FmtExpiry(in3h, now); !strings.Contains(got, "in 3h 0m") {
		t.Errorf("future: got %q", got)
	}
	ago := now.Add(-48 * time.Hour).UnixMilli()
	if got := FmtExpiry(ago, now); !strings.Contains(got, "2d 0h ago") {
		t.Errorf("past: got %q", got)
	}
	if got := FmtExpiry(0, now); got != "unknown" {
		t.Errorf("zero: got %q", got)
	}
}

func TestTable(t *testing.T) {
	out := Table([]string{"A", "BB"}, [][]string{{"x", "y"}, {"longer", "z"}})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "A") || !strings.Contains(lines[0], "BB") {
		t.Errorf("header misaligned: %q", lines[0])
	}
	idx1 := strings.Index(lines[1], "y")
	idx2 := strings.Index(lines[2], "z")
	if idx1 != idx2 {
		t.Errorf("column misaligned: %d != %d\n%s", idx1, idx2, out)
	}
}
