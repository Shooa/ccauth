// Package limits queries Anthropic's OAuth usage endpoint for rate-limit
// utilization (GET /api/oauth/usage). Zero cost: no inference is performed.
package limits

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const usageURL = "https://api.anthropic.com/api/oauth/usage"

type Bucket struct {
	Utilization float64   `json:"utilization"` // 0-100
	ResetsAt    time.Time `json:"resets_at"`
}

type Usage struct {
	FiveHour *Bucket `json:"five_hour"`
	SevenDay *Bucket `json:"seven_day"`
}

// Fetch returns current usage for an OAuth access token. The context bounds
// the whole request; pass a timeout context.
func Fetch(ctx context.Context, accessToken string) (Usage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usageURL, nil)
	if err != nil {
		return Usage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ccauth")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Usage{}, fmt.Errorf("usage: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusUnauthorized:
		return Usage{}, fmt.Errorf("usage: token expired or invalid")
	case http.StatusForbidden:
		return Usage{}, fmt.Errorf("usage: forbidden for this account")
	default:
		return Usage{}, fmt.Errorf("usage: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return Usage{}, fmt.Errorf("usage: read: %w", err)
	}
	var u Usage
	if err := json.Unmarshal(body, &u); err != nil {
		return Usage{}, fmt.Errorf("usage: parse: %w", err)
	}
	return u, nil
}

// Format renders a bucket as "27% (resets 14:19)"; empty when nil.
func Format(b *Bucket, now time.Time) string {
	if b == nil {
		return "-"
	}
	pct := int(b.Utilization + 0.5)
	s := fmt.Sprintf("%d%%", pct)
	if !b.ResetsAt.IsZero() {
		if pct >= 75 {
			s += fmt.Sprintf(" resets %s", b.ResetsAt.Local().Format("15:04"))
		}
	}
	return s
}
