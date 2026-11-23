# cpa-commandcode-quota

A [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) (CPA) 8.x plugin that shows your
[Command Code](https://commandcode.ai) plan limits on the management panel's **Quota** page, next to Claude,
Antigravity and Codex:

- plan name and monthly credits left (USD)
- 5-hour, weekly and monthly bars with % remaining and reset times

It is a quota-only plugin: it adds no models, executors or routes, and it never sends prompts anywhere.

## Requirements

- CLIProxyAPI 8.x with plugins enabled (the build with plugin support, not `_no-plugin`).
- The Command Code CLI logged in **as the user CPA runs as**. The plugin reads the API key from
  `~/.commandcode/auth.json` on every refresh, so it follows `cmd` re-logins. The key is only sent to
  `https://api.commandcode.ai`.
- Linux amd64/arm64 (glibc 2.28+) or Windows amd64 release builds. On macOS, build it yourself (below).

## Install

1. Add this repo as a plugin store source in `config.yaml` and restart CPA:

   ```yaml
   plugins:
     enabled: true
     store-sources:
       - "https://raw.githubusercontent.com/jimbb/cpa-commandcode-quota/main/registry.json"
   ```

   Then install **Command Code Quota** from the panel's **Plugins** page. It is enabled on install and
   loads without another restart.

   Manual install instead: download `commandcode-quota_<version>_<os>_<arch>.zip` from
   [Releases](https://github.com/jimbb/cpa-commandcode-quota/releases), check it against `checksums.txt`, put
   `commandcode-quota.so` into `plugins/<os>/<arch>/` next to the CPA binary, add
   `plugins.configs.commandcode-quota.enabled: true` to `config.yaml` and restart CPA.

2. The Quota page only lists auth files, so add a placeholder one to CPA's auth directory
   (`oauth.auth-dir`), e.g. `command-code.json`:

   ```json
   {"type": "command-code", "label": "Command Code"}
   ```

   Nothing in CPA is registered under `command-code`, so this file never serves requests. Keep the type
   `command-code`: if you also route Command Code through an `openai-compatibility` entry named `commandcode`,
   a `commandcode` placeholder would share that provider name and CPA would try it first for those models.

3. Open the panel's **Quota** page and press **Refresh quota** on the card.

## Notes

- The data comes from the endpoints the Command Code CLI itself uses for its usage view
  (`/alpha/billing/credits` and `/alpha/billing/subscriptions`). They are not a documented public API and
  may change.
- Personal accounts only: team plans would need the `orgId` parameter, which isn't sent yet.

## Build

```sh
go test ./...
go build -trimpath -buildmode=c-shared -o commandcode-quota.so .   # native build (also macOS: .dylib)
./release.sh 0.2.0   # release zips + checksums.txt in dist/ (needs Zig as the cgo cross-compiler)
```

`CC_LIVE=1 go test -run Live -v` makes one real call with your Command Code login.

## License

MIT
