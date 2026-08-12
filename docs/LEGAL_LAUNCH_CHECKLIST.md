# FormForge Legal Launch Checklist

This checklist separates code completed in 1.8.0 from operator actions that cannot be completed by source code alone.

## Before inviting paid users

- [ ] Decide whether to operate initially as a sole proprietor or form an entity.
- [ ] File the appropriate Tarrant County assumed-name/DBA record before doing business as FormForge.
- [ ] Do not launch paid under FormForge until a professional trademark clearance search is complete or a more distinctive replacement name is selected; an active fitness business already uses FormForge publicly.
- [ ] Obtain a free EIN if desired for banking and vendor forms.
- [ ] Open a separate bank account and bookkeeping ledger.
- [ ] Ask a Texas tax professional whether and where to collect sales tax; configure store tax settings.
- [ ] Obtain appropriate cyber/privacy, technology E&O, and general-liability quotes.
- [ ] Create a domain, real support inbox, privacy inbox, and moderation inbox.
- [ ] Fill every `FORMFORGE_*` legal environment variable in Render.
- [ ] Have a Texas attorney review the completed Terms, Privacy Notice, Community Standards, Subscription Terms, Health/AI Notice, deletion flow, and processor list.
- [ ] Decide with counsel whether to add arbitration, venue, class-action waiver, liability caps, indemnity, or special state notices.
- [ ] Execute and retain agreements/terms with Render, OpenAI, Apple, Google, email providers, storage providers, and any analytics/crash services actually enabled.

## App-store submission

- [ ] Implement Apple StoreKit and Google Play Billing; do not place API keys in the app.
- [ ] Verify subscriptions server-side and implement restore purchases.
- [ ] Add links to manage/cancel subscriptions in account settings.
- [ ] Complete Apple App Privacy and Google Data Safety disclosures so they match production behavior.
- [ ] Complete Google health-app and content-rating declarations accurately.
- [ ] Supply reviewer/demo credentials with stable backend access.
- [ ] Publish the privacy policy, terms, support, and account-deletion URLs on the final domain.
- [ ] Test report, block, moderation, account deletion, data export, and consent renewal on physical iOS and Android devices.

## Ongoing operations

- [ ] Review moderation reports on a defined schedule and document response times.
- [ ] Keep an appeals/contact process available.
- [ ] Review access logs, active sessions, backup integrity, and failed background jobs.
- [ ] Rotate credentials and test recovery regularly.
- [ ] Maintain an incident-response and health-data breach-notification process.
- [ ] Update policy versions and obtain renewed consent when changes are material.
- [ ] Preserve transaction/tax records according to professional advice while minimizing health-data retention.
