package main

import (
	"encoding/json"
	"os"
	"testing"
)

// CC_LIVE=1 go test -run Live -v: one real call to the Command Code API.
func TestLive(t *testing.T) {
	if os.Getenv("CC_LIVE") == "" {
		t.Skip("set CC_LIVE=1")
	}
	q, err := fetchQuota()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.MarshalIndent(q, "", "  ")
	t.Log(string(b))
}
