# FormForge 1.8.0 Test Report

## Executed successfully

- `gofmt`, `go test ./...`, and `go vet ./...`.
- JavaScript syntax checks for every `web/*.js` file, including the 1.8 legal/moderation layer and service worker.
- Chromium runtime smoke test for eighteen screens: Dashboard, Coaching Team, AI Coach, AI Agent, Workouts, Nutrition, Health + Plans, Progress, Habits, Community, Check-In, Marketplace, Mobile, Security, Administration, Appearance, More, and Settings.
- AI Coach message submission and phone-offline coach fallback.
- FormForge 1.8 public legal pages, hosted consent enforcement, policy versioning, adult age gate, account deletion, reports, user blocks, moderation filtering, user/content reporting, suspension, ban, and audit behavior.
- FormForge 1.7 focused navigation, complete More directory, four-card command center, active-workout controls, responsive shell, and reduced-motion-safe reveal behavior.
- Appearance setup, per-user preferences, five presets, custom theme persistence, focused/full navigation state, and administrator member-preview routing.
- Imperial/Metric setup, profile and goal conversion, progress display/input conversion, workout-load conversion, and canonical kg/cm persistence.
- Forge Knowledge Vault count, domain list, topic retrieval, knowledge-status endpoint, approved-source counts, and verified-quote counts.
- Existing FormForge 1.5 regression coverage: accounts, TOTP, sessions, protected exports, database migrations, backups, health states, creator imports/takedowns, AI usage caps, source grounding, workouts, RPE progression, pain substitutions, nutrition, health imports, encrypted photos, meal plans, social features, signed updates, autonomous-agent task storage, memories, and marketplace foundations.
- Live cloud-mode HTTP startup on port 18080 with schema version 9, `status: up`, and successful rendering of Privacy and Account Deletion pages.
- Linux x86-64 build and executable-format/SHA-256 inspection. Previous release coverage included Windows x64 and Linux ARM64 cross-compilation; those physical targets were not re-executed for this Render-focused legal update.

## Not executed in this environment

- Launching the executables on a physical Windows 10/11 machine, registry/shortcut behavior, firewall prompts, Edge app-mode behavior, update installer handoff, and uninstall UI.
- Physical iPhone/iPad/Android PWA installation, local CA trust workflow, camera scanning, and real Bluetooth/vision hardware.
- Live vendor OAuth, payment processing, production crash-report hosting, or production signed-update hosting because credentials and infrastructure were not supplied.
- A live paid model request or exhaustive factual/medical validation of every possible AI answer.
- External penetration testing, legal review, medical review, creator licensing review, and app-store review.
