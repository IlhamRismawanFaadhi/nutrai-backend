package service

import (
	"encoding/json"
	"math"
	"strings"
)

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }

func pct(v, target float64) int {
	if target <= 0 {
		return 0
	}
	return int(math.Round(v / target * 100))
}

func clamp0(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

// normalizeTags: lowercase, trim, dedupe, buang kosong. Selalu non-nil.
func normalizeTags(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || len(t) > 40 || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func tagsToJSON(in []string) string {
	b, _ := json.Marshal(normalizeTags(in))
	return string(b)
}

func jsonToTags(s string) []string {
	out := []string{}
	if s != "" {
		_ = json.Unmarshal([]byte(s), &out)
	}
	if out == nil {
		out = []string{}
	}
	return out
}

// Slugify: "Telur dadar & roti gandum" -> "telur-dadar-roti-gandum" (harus sama dengan docs/food-images-manifest.csv).
func Slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
