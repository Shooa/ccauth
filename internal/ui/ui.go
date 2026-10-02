package ui

import (
	"fmt"
	"strings"
	"time"
)

func HumanDur(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d >= 365*24*time.Hour:
		return fmt.Sprintf("%.1fy", d.Hours()/(24*365))
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24)
	case d >= time.Hour:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	case d >= time.Minute:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}

// FmtExpiry renders a millisecond epoch as "2006-01-02 15:04 (in 3h 12m)".
func FmtExpiry(ms int64, now time.Time) string {
	if ms == 0 {
		return "unknown"
	}
	t := time.UnixMilli(ms).Local()
	d := t.Sub(now)
	rel := fmt.Sprintf("in %s", HumanDur(d))
	if d < 0 {
		rel = fmt.Sprintf("%s ago", HumanDur(d))
	}
	return fmt.Sprintf("%s (%s)", t.Format("2006-01-02 15:04"), rel)
}

func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return s[:n-1] + "…"
}

// Table renders rows with dynamic column widths.
func Table(headers []string, rows [][]string) string {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if i < len(widths) && len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	var b strings.Builder
	line := func(cells []string) {
		var parts []string
		for i, c := range cells {
			parts = append(parts, fmt.Sprintf("%-*s", widths[i], c))
		}
		b.WriteString(strings.TrimRight(strings.Join(parts, "  "), " "))
		b.WriteString("\n")
	}
	line(headers)
	for _, r := range rows {
		line(r)
	}
	return b.String()
}
