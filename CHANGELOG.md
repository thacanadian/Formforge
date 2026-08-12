# Changelog

## 1.8.0 — 2026-08-04

### Legal and privacy
- Added public Terms, Privacy, Community Standards, Subscription, Health/AI, Security/Breach, and account-deletion pages.
- Added configurable operator identity, support/privacy/moderation contacts, mailing address, minimum age, price, and governing-law wording through environment variables.
- Added versioned Terms, Privacy, Community, and Subscription policy records with user acceptance timestamps and renewed-consent support.
- Enforced 18+ hosted setup and explicit hosted Terms/Privacy acceptance.
- Added in-app and public web account deletion with active-data/photo removal, session invalidation, audit anonymization, and sole-administrator protection.

### Community moderation
- Added report flows for nudges, shared workouts, and user accounts.
- Added user blocking, reporter-side content hiding, text safety filtering, spam checks, and nudge rate limiting.
- Added an administrator report queue with reported-content context, dismiss/remove/warn/suspend/ban actions, restriction reasons, and moderation audit records.
- Added published Community Standards and a human moderation operations guide.

### Compliance and validation
- Added store-compliance and sole-proprietor launch checklists plus attorney-review copies of every legal page.
- Updated Render Blueprint variables for production legal identity and contact configuration.
- Added schema migration 9 and regression tests for legal pages, consent gates, reports, blocks, suspension, deletion, and hosted setup enforcement.
- Passed the complete Go test suite, Go vet, JavaScript syntax validation, and Chromium smoke tests across all application screens.

## 1.7.1 — Render beta deployment

- Added `--cloud` mode for managed reverse proxies: plain internal HTTP, `PORT` support, `0.0.0.0` binding, Render HTTPS/HSTS, and no local certificate requirement.
- Added trusted-proxy client-IP handling for accurate login throttling, sessions, and audit records behind Render.
- Added a generated `FORMFORGE_SETUP_TOKEN` gate so the first public visitor cannot claim the administrator account.
- Added optional environment configuration for the OpenAI API key, model, AI mode, autonomous agent, SearXNG endpoint, and takedown contact.
- Added hosted-mobile instructions that install directly from the Render HTTPS address without a local CA certificate.
- Added `render.yaml`, `.env.render.example`, and `docs/RENDER_DEPLOYMENT.md`.
- Preserved local Windows, LAN, Raspberry Pi, and offline-coach behavior.

## 1.7.0 — 2026-08-03

### Focused command center
- Rebuilt the primary shell around a calmer retro-futuristic terminal aesthetic with restrained amber glow, larger typography, clearer spacing, and fewer competing panels.
- Replaced the crowded dashboard with four primary cards: Today’s Workout, Recovery Score, AI Coach, and Nutrition Summary.
- Added responsive desktop and mobile layouts with a five-destination mobile navigation bar.
- Kept the complete feature set available through a redesigned More screen instead of removing capabilities.

### Motion and interaction
- Added reduced-motion-safe scroll reveals, page transitions, subtle hover depth, a reading-progress indicator, and status feedback.
- Added a focused active-workout session with a timer, exercise completion controls, and handoff to the existing detailed workout logger.
- Added direct dashboard actions for starting a workout, opening recovery and nutrition, and asking the coach to adjust the day’s plan.

### Navigation and access
- Simplified focused desktop navigation to Dashboard, AI Coach, Workouts, Nutrition, Progress, and More.
- Preserved Coaching Team, FormForge Agent, Health + Recovery, Habits, Check-In, Community, Marketplace, Mobile setup, Appearance, Security, Profile/Data, and administrator tools under More.
- Preserved optional full navigation and administrator member-preview behavior.

### Validation
- Added FormForge 1.7 assets to the service worker and Windows/Linux build checks.
- Passed Go tests and vet, JavaScript syntax checks, and the Chromium runtime smoke suite across all eighteen application screens.

## 1.6.0 — 2026-07-25

### Appearance and measurements
- Moved appearance controls out of the crowded Settings page into a dedicated Appearance screen and top-bar shortcut.
- Added Forge, Midnight, Iron, Arctic, and Forest presets plus custom colors, density, and corner-radius controls.
- Added per-user Imperial and Metric modes across onboarding, profile editing, progress, workout loads, goal weights, and offline coaching. Canonical values remain stored in kilograms and centimeters.

### Member experience
- Added focused standard-user navigation with a More page that preserves access to all authorized features.
- Added optional full user navigation.
- Added administrator-only Preview member view without changing the active account or permissions.

### AI knowledge, quotes, and links
- Preserved the complete offline/online/Auto Coach and local autonomous Agent feature set.
- Added the Forge Knowledge Vault with 60 original expert reference modules across training, nutrition, recovery, evidence, and safety.
- Added retrieval from the local vault, user history, approved creator sources, verified quotes, and optional current public web research.
- Added explicit source/link-request detection and real-URL responses in online/agent mode.
- Tightened exact-quotation handling so unverified wording is paraphrased rather than fabricated.
- Added a knowledge-status endpoint showing local modules, domains, approved sources, and verified quotations.

### Testing and packaging
- Expanded browser runtime coverage to eighteen screens, including Appearance and More.
- Added tests for setup preferences, unit persistence/conversion, knowledge retrieval, and status reporting.
- Added Windows, portable, Raspberry Pi ARM64, source, and clean-distribution release packages.

## 1.5.0 — 2026-07-24

### Security and data integrity
- Added optional TOTP two-factor authentication and recovery codes for administrator accounts.
- Added per-device/session visibility and manual revocation.
- Added AES-GCM password-protected JSON and CSV exports and protected JSON import.
- Added schema migrations with migration history and pre-migration safety copies during startup and restore.
- Added `up`, `degraded`, and `down` server health reporting.
- Added per-user/day online-AI token and estimated-cost accounting and caps.
- Added response grounding labels, individual chat deletion, and terms acceptance.
- Added structured local logs, client error logging, server panic records, and opt-in crash reporting.
- Added background backup and integrity jobs.
- Extended encrypted backups to include encrypted progress-photo files.

### Coaching Team
- Expanded the built-in influence roster to more than twenty editorial profiles.
- Added public-link imports for YouTube, TikTok, Instagram, Spotify, podcasts, Threads, X, Facebook, newsletters, and websites.
- Added platform detection, public metadata previews, duplicate checks, private-network URL rejection, custom profile editing/removal, and Official/Editorial status.
- Added preferred-coach tiebreaking.
- Added creator takedown submission, review, resolution, and global profile removal.

### Training, health, and nutrition
- Added exercise weight/reps/sets/RPE logging and automatic progressive-overload suggestions.
- Added pain flags and pain-aware workout substitutions.
- Added encrypted local progress photos.
- Added 1–14 day meal planning.
- Added Apple Health, Google Fit/Health Connect, Garmin, WHOOP, Oura, and HR-strap data imports.
- Added compatible-browser Web Bluetooth heart-rate capture.

### Social and commercial controls
- Added household leaderboards, nudges, and shared workout scheduling.
- Added Free and Pro plan assignment; Free remains offline-first and Pro can use capped online AI.
- Added terms covering Editorial creator profiles, right-of-publicity boundaries, source verification, and takedown handling.

### Infrastructure
- Added signed update manifests with Ed25519 and SHA-256 verification.
- Added an update-manifest signing utility.
- Expanded mobile and browser runtime smoke tests to all fourteen screens.

## 1.3.0 — 2026-07-24
- Added Coaching Team blends, editorial creator profiles, source library, and verified quotes.

## 1.2.0 — 2026-07-24
- Added mobile PWA installation, LAN setup, startup repair, and phone-offline coaching.

## 1.1.0 — 2026-07-24
- Added offline/online AI Coach and corrected empty-list page crashes.

## 1.0.0 — 2026-07-24
- Initial Windows, portable, HTTPS, multi-user, fitness-tracking, and encrypted-backup release.
