# FormForge 1.4 Development and Build Guide

## Requirements

- Go 1.22+
- Optional Node.js for JavaScript syntax checks
- Optional Chromium plus Python `requests` and `websocket-client` for the UI runtime smoke test

## Validate

```bash
gofmt -w ./internal/core ./main.go
go test ./...
go vet ./...
node --check web/app.js
node --check web/v14.js
node --check web/coaching-ui.js
node --check web/sw.js
```

Browser smoke test:

```bash
chromium --headless=new --no-sandbox --remote-debugging-port=9223 --remote-allow-origins='*' --user-data-dir=/tmp/formforge-ui about:blank
python3 scripts/test-ui-runtime.py
```

## Build all Windows artifacts on Linux/macOS

```bash
./scripts/build-all.sh
```

## Build on Windows

```powershell
.\scripts\build-windows.ps1
```

The build generates a Windows x64 GUI executable, uninstaller, self-contained bootstrap Setup.exe, and a Linux smoke-test executable. The installer copies application binaries and documentation into `%LOCALAPPDATA%\Programs\FormForge` and leaves `%LOCALAPPDATA%\FormForge` untouched.

## Database migrations

Increment `SchemaVersion`, append exactly one sequential migration to `migrations.go`, and add tests for opening/restoring older schemas. Never skip a version. Startup creates a migration safety copy before applying changes.

## Signed updates

```bash
go run ./tools/update-signer -generate-key
go run ./tools/update-signer -version 1.4.1 -url https://updates.example.com/FormForge-Setup-1.4.1.exe -file FormForge-Setup-1.4.1.exe -private-key update-private.key -out update-manifest.json
```

Host the manifest and installer on HTTPS. Configure FormForge with the manifest URL and base64 public key. Keep the private key offline.
