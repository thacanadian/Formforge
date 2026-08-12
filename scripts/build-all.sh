#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version="$(tr -d '\r\n' < VERSION)"
mkdir -p dist installer/bootstrap/payload
rm -f installer/bootstrap/payload/*
printf '%s\n' 'Build scripts replace this placeholder with the verified installer payload.' > installer/bootstrap/payload/BUILD_PAYLOAD.txt
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -H=windowsgui -X main.version=$version" -o dist/FormForge.exe .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -H=windowsgui" -o dist/Uninstall.exe ./installer/uninstaller
rm -f installer/bootstrap/payload/BUILD_PAYLOAD.txt
cp dist/FormForge.exe installer/bootstrap/payload/FormForge.exe
cp dist/Uninstall.exe installer/bootstrap/payload/Uninstall.exe
cp README.md VERSION CHANGELOG.md LICENSE.txt installer/formforge.ico docs/*.md scripts/enable-lan-firewall.ps1 scripts/disable-lan-firewall.ps1 installer/bootstrap/payload/
go test ./...
go vet ./...
command -v node >/dev/null && node --check web/app.js && node --check web/v14.js && node --check web/v15.js && node --check web/v16.js && node --check web/v17.js && node --check web/coaching-ui.js && node --check web/sw.js || true
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -H=windowsgui" -o dist/Setup.exe ./installer/bootstrap
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$version" -o dist/formforge-linux-test .
