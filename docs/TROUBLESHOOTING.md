# FormForge 1.4 Troubleshooting

## “Failed to fetch” or “could not connect”

1. Close old browser-installed localhost FormForge windows.
2. Launch the real FormForge Desktop/Start Menu shortcut so `FormForge.exe` starts the backend.
3. Install 1.8.0 over the current version if the shortcut or repair protocol is missing.
4. Use the connection-repair page’s retry/start controls.
5. Check `%LOCALAPPDATA%\FormForge\logs\events.jsonl` and the console log.
6. Visit `/health`; `degraded` includes specific issues, while `down` returns HTTP 503.

## Mobile cannot connect

Confirm LAN access is enabled, the PC and phone are on the same private Wi-Fi, FormForge is running, the private firewall rule is enabled, and the phone trusts the local CA. Guest networks may block device-to-device traffic.

## TOTP login failure

Confirm the phone clock is automatic and enter the current six-digit code. A saved recovery code can be used once in the same login field. Administrator recovery with the installation recovery key can reset the password, but disabling TOTP through the app requires password and current code.

## Online AI unavailable

Check the user’s Free/Pro tier, daily usage caps, API-key configuration, provider account, internet access, model name, and service URL. Auto mode falls back offline. Local accounting is an estimate; provider quotas and bills remain authoritative.

## Wearable import fails

Use the provider’s export format listed in `WEARABLES.md`. CSV must contain a header and fields such as metric/type, date/startAt, value, unit, and source. Apple Health uses the exported XML file. Direct vendor OAuth is not included.

## Backup or restore failure

Confirm the recovery key matches the backup, the file is not truncated, and there is free disk space for the database and photos. Restore runs migrations before replacement and signs out every user.

## Update not configured

Signed updates require an HTTPS manifest URL and Ed25519 public key in administrator settings. Use `tools/update-signer` to generate/sign a channel. Never configure the private key in FormForge.

## Repair installation

Run `FormForge-Setup-1.8.0.exe` again. The installer verifies embedded payloads and preserves `%LOCALAPPDATA%\FormForge`. A normal uninstall also preserves user data unless the operator manually deletes that folder.
