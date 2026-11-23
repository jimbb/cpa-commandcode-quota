// Command Code quota for the CPA (CLIProxyAPI) management panel.
//
// A quota-only CPA plugin: an auth file with "type": "command-code" gets a card on the panel's Quota page
// with the 5-hour, weekly and monthly Command Code limits. The panel's Quota page lists auth files only,
// so config credentials (e.g. an openai-compatibility entry) can't carry the card. No executor or model
// is registered under "command-code", so that auth file never routes traffic. Don't use "commandcode":
// an openai-compatibility entry with that name registers its models under it and CPA would then try
// the placeholder for them first (401, then fall through).
// The API key is read from the Command Code CLI's own login (~/.commandcode/auth.json), so it follows
// `cmd` re-logins.
// Build: see README.md and release.sh.
package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef struct cliproxy_host_api cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"unsafe"
)

var version = "0.1.0" // -ldflags "-X main.version=..."

const apiBase = "https://api.commandcode.ai"

const provider = "command-code" // auth file "type"

// Monthly credits (USD) and display names per plan, from the Command Code CLI's getPlanTotalCredits.
var plans = map[string]struct {
	name    string
	credits float64
}{
	"individual-go": {"Go", 10}, "individual-go-v1": {"Go", 10}, "individual-goat": {"GOAT", 70},
	"individual-pro": {"Pro", 30}, "individual-pro-v1": {"Pro", 80}, "individual-provider": {"Provider", 15},
	"individual-max": {"Max", 150}, "individual-ultra": {"Ultra", 300}, "teams-pro": {"Teams Pro", 40},
}

type window struct {
	Used    float64 `json:"used"`
	Cap     float64 `json:"cap"`
	ResetAt int64   `json:"resetAt"`
}

type credits struct {
	Credits struct {
		MonthlyCredits   float64 `json:"monthlyCredits"`
		PurchasedCredits float64 `json:"purchasedCredits"`
	} `json:"credits"`
	WindowLimits struct {
		FiveHour *window `json:"fiveHour"`
		Weekly   *window `json:"weekly"`
	} `json:"windowLimits"`
}

type subscription struct {
	Data *struct {
		PlanID           string `json:"planId"`
		Status           string `json:"status"`
		CurrentPeriodEnd string `json:"currentPeriodEnd"`
	} `json:"data"`
}

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	plugin.abi_version = 1
	plugin.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	result, err := handle(C.GoString(method))
	env, rc := map[string]any{"ok": true, "result": result}, C.int(0)
	if err != nil {
		env, rc = map[string]any{"ok": false, "error": map[string]string{"code": "plugin_error", "message": err.Error()}}, 1
	}
	raw, _ := json.Marshal(env)
	if response != nil {
		response.ptr = C.CBytes(raw)
		response.len = C.size_t(len(raw))
	}
	return rc
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, size C.size_t) { C.free(ptr) }

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {}

func handle(method string) (any, error) {
	switch method {
	case "plugin.register", "plugin.reconfigure":
		return map[string]any{
			"schema_version": 10,
			"metadata": map[string]any{"Name": "Command Code Quota", "Version": version, "Author": "jimbb",
				"GitHubRepository": "https://github.com/jimbb/cpa-commandcode-quota"},
			"capabilities": map[string]any{"quota_provider": true},
		}, nil
	case "quota.identifier":
		return map[string]string{"identifier": provider}, nil
	case "quota.describe":
		return map[string]any{"display_name": "Command Code", "supported_providers": []string{provider}}, nil
	case "quota.fetch":
		return fetchQuota()
	}
	return nil, fmt.Errorf("unsupported method %s", method)
}

func fetchQuota() (any, error) {
	home, _ := os.UserHomeDir()
	raw, err := os.ReadFile(filepath.Join(home, ".commandcode", "auth.json"))
	if err != nil {
		return nil, fmt.Errorf("Command Code login not found: %w", err)
	}
	var auth struct {
		APIKey string `json:"apiKey"`
	}
	if json.Unmarshal(raw, &auth); auth.APIKey == "" {
		return nil, fmt.Errorf("no apiKey in ~/.commandcode/auth.json: run `cmd` and log in")
	}
	// ponytail: personal accounts only; team plans need ?orgId= from /alpha/whoami
	var c credits
	var s subscription
	if err := get(auth.APIKey, "/alpha/billing/credits", &c); err != nil {
		return nil, err
	}
	if err := get(auth.APIKey, "/alpha/billing/subscriptions", &s); err != nil {
		return nil, err
	}
	return buildQuota(c, s), nil
}

var client = &http.Client{Timeout: 20 * time.Second}

func get(key, path string, out any) error {
	req, _ := http.NewRequest(http.MethodGet, apiBase+path, nil)
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("%s: HTTP %d %s", path, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// buildQuota maps Command Code billing data onto CPA's normalized quota shape.
func buildQuota(c credits, s subscription) map[string]any {
	var buckets []map[string]any
	limit := func(win, label string, w *window) {
		if w == nil || w.Cap <= 0 {
			return
		}
		buckets = append(buckets, map[string]any{
			"window":            win,
			"remainingFraction": clamp(1 - w.Used/w.Cap),
			"resetTime":         time.UnixMilli(w.ResetAt).UTC().Format(time.RFC3339),
			"description":       label, // the panel uses it as the row label, which fits ~7 chars
		})
	}
	limit("5h", "5-hour", c.WindowLimits.FiveHour)
	limit("weekly", "Weekly", c.WindowLimits.Weekly)

	out := map[string]any{}
	summary := []map[string]any{money("monthly_credits", "Monthly credits left", c.Credits.MonthlyCredits)}
	if c.Credits.PurchasedCredits > 0 {
		summary = append(summary, money("purchased_credits", "Purchased credits", c.Credits.PurchasedCredits))
	}
	if d := s.Data; d != nil {
		p, ok := plans[d.PlanID]
		name := p.name
		if !ok {
			name = d.PlanID
		}
		if d.Status != "active" {
			name += " (" + d.Status + ")"
		}
		out["subscription"] = map[string]any{"plan": name, "tierId": d.PlanID}
		if p.credits > 0 {
			buckets = append(buckets, map[string]any{
				"window":            "30d",
				"remainingFraction": clamp(c.Credits.MonthlyCredits / p.credits),
				"resetTime":         d.CurrentPeriodEnd,
				"description":       "Monthly",
			})
		}
	}
	out["summary"] = summary
	out["groups"] = []map[string]any{{"displayName": "Command Code", "buckets": buckets}}
	return out
}

func money(key, label string, v float64) map[string]any {
	return map[string]any{"key": key, "label": label, "value": v, "format": "currency", "currency": "USD"}
}

func clamp(f float64) float64 { return max(0, min(1, f)) }
