# FormForge 1.4 Security Guide

FormForge is designed for a small trusted household or team on a private computer/network. It is not hardened or independently audited as a public multi-tenant SaaS product.

## Authentication

Passwords are stored using PBKDF2-HMAC-SHA-256 with random salts and a high iteration count. Sessions use random tokens stored as hashes, HTTP-only secure cookies, strict SameSite behavior, 12-hour expirations, CSRF tokens for state-changing requests, failed-login throttling, and temporary lockouts. Administrator accounts can enable RFC 6238 TOTP through the **Security** page and receive one-time recovery codes. Each signed-in device/session is visible and can be revoked.

## Network exposure

FormForge binds only to `127.0.0.1` by default. LAN access is opt-in and should be limited to Windows private-network profiles. HTTPS uses a locally generated certificate authority. Every approved mobile device must trust that CA. Never expose the FormForge port directly to the public internet. Use a private VPN or properly managed reverse proxy/hosting for remote access.

## AI privacy and spending controls

The offline coach does not call an external provider. Online mode sends the current prompt, limited recent chat context, profile summary, selected Coaching Team principles, and approved source summaries to the configured provider. The API key is AES-GCM encrypted with the local master key and never returned to browser JavaScript. Free/Pro permissions and daily token/cost caps reduce accidental API spending. Cost values are conservative local estimates and are not a replacement for the provider’s billing dashboard.

## Health and photo data

Health imports and progress-photo metadata are stored in the local database. Photo bytes are separately AES-GCM encrypted under `%LOCALAPPDATA%\FormForge\photos`. FormForge does not upload photos. Encrypted backups include the database and encrypted photo blobs. Anyone who controls the Windows account and local master key can operate FormForge; protect the Windows account and disk with strong login credentials and full-disk encryption.

## Exports and backups

Normal JSON/CSV exports are readable files. Use the password-protected export controls for sensitive data; these use PBKDF2-derived AES-GCM encryption plus an integrity hash. Backups use the installation recovery key and AES-GCM. Store the recovery key separately from backups. Clean distributions must never contain `formforge.db.json`, `master.key`, certificates, logs, backups, photo files, API keys, or recovery codes.

## Structured logs and crash reporting

Local events are appended to `logs/events.jsonl`. Do not share logs without reviewing them for usernames, IPs, paths, or error context. Crash reporting is disabled by default and can only target an administrator-configured HTTPS endpoint. No telemetry endpoint is bundled.

## Signed updates

A configured update channel must use HTTPS, a base64 Ed25519 public key, a signed manifest, and an installer SHA-256 hash. FormForge verifies the manifest signature and downloaded installer hash before staging. Keep the private signing key offline and never ship it with the application.

## Creator profiles

Editorial profiles are summaries, not impersonations or endorsements. Do not add protected names, likenesses, exact voices, trademarks, paid programs, complete transcripts, or copyrighted material without a lawful basis. Exact quotations require a verified source. Follow `CREATOR_TAKEDOWN.md` for credible complaints.

## Remaining limitations

- The live record store is an atomic JSON database rather than a high-concurrency SQL server.
- Live database records are protected by OS permissions but are not field-encrypted at rest.
- The local certificate authority creates trust responsibility for the installation owner.
- The application has not undergone an external penetration test.
- Direct vendor wearable OAuth is not bundled; file imports and supported-browser HR Bluetooth are local connectors.
- Commercial operators need legal, privacy, medical-disclaimer, tax, billing, and creator-license review for their jurisdiction.
