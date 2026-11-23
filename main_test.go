package main

import (
	"encoding/json"
	"testing"
)

func TestBuildQuota(t *testing.T) {
	var c credits
	var s subscription
	json.Unmarshal([]byte(`{"credits":{"monthlyCredits":6.5,"purchasedCredits":0},
		"windowLimits":{"fiveHour":{"used":0.75,"cap":3,"resetAt":1795432130032},"weekly":{"used":7,"cap":6,"resetAt":1795745052821}}}`), &c)
	json.Unmarshal([]byte(`{"data":{"planId":"individual-go-v1","status":"active","currentPeriodEnd":"2026-12-12T14:46:05.000Z"}}`), &s)

	q := buildQuota(c, s)
	b := q["groups"].([]map[string]any)[0]["buckets"].([]map[string]any)
	want := []float64{0.75, 0, 0.65} // 5h 25% used, weekly over cap, $6.50 of $10 left
	if len(b) != len(want) {
		t.Fatalf("buckets = %v", b)
	}
	for i, w := range want {
		if got := b[i]["remainingFraction"].(float64); got != w {
			t.Errorf("bucket %s remaining = %v, want %v", b[i]["window"], got, w)
		}
	}
	if plan := q["subscription"].(map[string]any)["plan"]; plan != "Go" {
		t.Errorf("plan = %v", plan)
	}
}
