# Wearable and Health Data Connections

FormForge 1.4 supports local import/sync workflows for:

- Apple Health export XML
- Google Fit or Health Connect JSON/CSV exports
- Garmin CSV/JSON exports
- WHOOP CSV/JSON exports
- Oura CSV/JSON exports
- Heart-rate strap CSV sessions
- Live Web Bluetooth heart-rate readings in compatible browsers

Open **Health + Plans**, select the provider and file format, and choose an export file. FormForge normalizes common metrics including steps, heart rate, resting heart rate, sleep, active calories, body weight, workouts, and recovery.

Direct background OAuth synchronization with Apple, Google, Garmin, WHOOP, or Oura is not included in the clean build because each provider requires separate developer registration, contracts, credentials, redirect URLs, privacy disclosures, and often platform-specific native entitlements. The architecture stores provider connections and normalized metrics so an authorized deployment can add those connectors without changing user-facing records. Do not place vendor client secrets in frontend JavaScript.

Web Bluetooth requires a browser/device that exposes the Heart Rate service. iOS browser support may differ from Chromium-based Android/desktop support; CSV import remains the fallback.
