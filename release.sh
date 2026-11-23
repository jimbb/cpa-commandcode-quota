#!/bin/sh
# Build the CPA plugin-store release assets into dist/: <id>_<version>_<os>_<arch>.zip (library at the
# zip root) + checksums.txt. Needs Go and Zig (used as the cgo cross-compiler; glibc 2.28 baseline).
# Publish: gh release create v<version> dist/*
set -eu
ID=commandcode-quota
VERSION=${1:?usage: ./release.sh <version>}
ZIG=${ZIG:-zig}
rm -rf dist && mkdir dist
for t in linux/amd64/x86_64-linux-gnu.2.28 linux/arm64/aarch64-linux-gnu.2.28 windows/amd64/x86_64-windows-gnu; do
	os=${t%%/*} rest=${t#*/}
	arch=${rest%%/*} target=${rest#*/}
	ext=so
	[ "$os" = windows ] && ext=dll
	work=$(mktemp -d)
	# -g0 + prefix map: no local paths (zig/toolchain dirs) end up in the C parts of the binary
	CGO_ENABLED=1 GOOS=$os GOARCH=$arch CC="$ZIG cc -target $target" CGO_CFLAGS="-O2 -g0 -ffile-prefix-map=$(dirname "$(command -v "$ZIG")")=zig" \
		go build -trimpath -buildvcs=false -buildmode=c-shared -ldflags "-s -w -X main.version=$VERSION" -o "$work/$ID.$ext" .
	chmod 755 "$work/$ID.$ext"
	(cd "$work" && python3 -m zipfile -c "$OLDPWD/dist/${ID}_${VERSION}_${os}_${arch}.zip" "$ID.$ext")
	rm -rf "$work"
done
(cd dist && sha256sum -- *.zip >checksums.txt)
