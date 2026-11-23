# cpa-commandcode-quota

CPA (CLIProxyAPI 8.x) plugin that shows Command Code plan limits (5-hour, weekly, monthly credits) on the
management panel's **Quota** page, next to Claude and Antigravity.

- Quota-only plugin (`quota_provider` capability), C ABI like the official plugins.
- Reads the API key from the Command Code CLI login, `~/.commandcode/auth.json`, of the user CPA runs as.
- Calls `https://api.commandcode.ai/alpha/billing/credits` and `/alpha/billing/subscriptions` (the CLI's `/usage` endpoints).
- The panel's Quota page lists auth files only, so it needs a placeholder auth file in CPA's auth dir:
  `{"type": "command-code", "email": "<you>", "label": "Command Code"}`. Nothing is registered under
  `command-code`, so it never routes traffic. Don't name the type `commandcode` if an openai-compatibility
  entry has that name: CPA would try the placeholder for that entry's models first.

## Build and install

```sh
go test ./... && go build -buildmode=c-shared -ldflags "-s -w -X main.version=0.2.0" -o commandcode-quota-v0.2.0.so .
cp commandcode-quota-v0.2.0.so ~/cliproxyapi/plugins/linux/amd64/
```

Enable it in `config.yaml` (`plugins.configs.commandcode-quota.enabled: true`) and restart CPA.
`CC_LIVE=1 go test -run Live -v` makes one real API call.
