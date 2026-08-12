# Deploy FormForge on Render

This build includes a hosted mode for a small private beta. Render terminates public HTTPS and forwards plain HTTP to FormForge. FormForge still uses secure cookies, CSRF protection, login throttling, encrypted API-key storage, encrypted backups, and the local JSON datastore.

## Required Render resources

Use one paid Render web service with one persistent disk mounted at `/var/data`.

Do not use a free web service for real trial data. Free services have an ephemeral filesystem and cannot attach a persistent disk, so accounts, logs, photos, encryption keys, and database changes can disappear after a restart or redeploy.

The included `render.yaml` defines:

- Go web service
- Starter instance
- Ohio region
- One instance
- `/health` health check
- 1 GB disk mounted at `/var/data`
- Cloud mode and secure first-admin setup token

## Deployment steps

1. Create a private GitHub repository and upload the contents of this source folder to the repository root.
2. In Render, choose **New → Blueprint** and select the repository.
3. Confirm the paid web-service plan and persistent disk.
4. Deploy the Blueprint.
5. Open **Environment** for the new service and copy the generated `FORMFORGE_SETUP_TOKEN` value.
6. Open the service's `https://<name>.onrender.com` address.
7. Paste the setup token into the first-admin setup form and create the administrator account immediately.
8. Save the one-time FormForge recovery key outside Render.
9. Before sharing the URL, fill the required legal environment variables listed below and redeploy.
10. Enable administrator TOTP before inviting friends.
11. Create separate standard-user accounts for each friend.

## Required legal and public-contact configuration

The app deliberately shows an administrator warning until these values are real. Never publish the placeholder legal pages.

```text
FORMFORGE_BUSINESS_NAME=FormForge
FORMFORGE_LEGAL_NAME=Your full legal name
FORMFORGE_ENTITY_DESCRIPTION=an individual sole proprietor
FORMFORGE_SUPPORT_EMAIL=support@your-domain.com
FORMFORGE_PRIVACY_EMAIL=privacy@your-domain.com
FORMFORGE_MODERATION_EMAIL=safety@your-domain.com
FORMFORGE_MAILING_ADDRESS=Your valid business mailing address
FORMFORGE_MINIMUM_AGE=18
FORMFORGE_MONTHLY_PRICE=$20.00 per month
FORMFORGE_GOVERNING_LAW=Texas, United States
FORMFORGE_TAKEDOWN_CONTACT=safety@your-domain.com
```

These values appear on `/legal`, the policy pages, and account-deletion page. Use a valid address and working inboxes. Do not use a fake identity or placeholder address. Review `docs/LEGAL_LAUNCH_CHECKLIST.md` before accepting paid users.

## Optional OpenAI connection

The app works without an external key by using the bundled offline coach.

For online coaching:

1. Create an OpenAI API project key in the OpenAI Platform.
2. In Render, open **Environment → Add Environment Variable**.
3. Add `OPENAI_API_KEY` as a secret and paste the key there.
4. Leave these values as configured by the Blueprint:
   - `FORMFORGE_AI_MODE=auto`
   - `FORMFORGE_AI_BASE_URL=https://api.openai.com/v1`
   - `FORMFORGE_AI_MODEL=gpt-4o-mini`
5. Redeploy or restart the service.
6. Sign in as administrator, open **AI Coach**, and use **Test online connection**.

`OPENAI_API_KEY` is read only by the Go backend. At startup it is encrypted using FormForge's persistent master key. It is not embedded in frontend JavaScript and is never returned by the settings API.

When `OPENAI_API_KEY` remains defined in Render, it is the source of truth and is reapplied at each restart. Remove it from Render before managing the key only through FormForge's administrator UI.

## Autonomous agent

The FormForge Agent uses the same OpenAI key as the AI Coach. It does not need a second model-provider key.

To enable model-only agent tasks, set:

```text
FORMFORGE_AGENT_ENABLED=true
FORMFORGE_AGENT_BASE_URL=https://api.openai.com/v1
FORMFORGE_AGENT_MODEL=gpt-4o-mini
```

Web research remains disabled until a compatible SearXNG endpoint is supplied:

```text
FORMFORGE_AGENT_ALLOW_WEB=true
FORMFORGE_AGENT_SEARCH_URL=https://your-searxng.example/search
```

The current SearXNG connector expects JSON search output and has no field for a search API key. Use a private or access-controlled SearXNG deployment rather than an unknown public instance.

## What does not require an API key

- Offline AI Coach and Forge Knowledge Vault
- Open Food Facts food and barcode lookup
- YouTube, TikTok, and Spotify public oEmbed metadata previews
- Health-export file imports
- Manual workouts, nutrition, progress, habits, and check-ins
- PWA installation
- FormForge encrypted backups
- Render-managed HTTPS

## Not connected in this build

These require separate future integration and credentials:

- Direct Apple Health, Health Connect, Garmin, WHOOP, or Oura background synchronization
- Stripe or another payment processor
- Email/SMS delivery
- Push-notification provider
- Error-monitoring service
- A hosted SearXNG service

## Updating

Push updates to the connected Git branch. Render rebuilds and restarts the service. Only `/var/data` persists, so keep all live data under that mount path. The persistent disk forces a brief restart window during deployment.

Before an update:

1. Open **More → Profile/Data → Recovery**.
2. Create and download an encrypted FormForge backup.
3. Push the update.
4. Confirm `/health` returns `status: up` or `degraded` rather than `down`.

## Security checklist for a friends-only beta

- Keep the GitHub repository private.
- Never commit `.env` files or API keys.
- Use the generated setup token before creating the first admin.
- Use unique 12+ character passwords.
- Enable TOTP for the administrator.
- Give friends standard accounts, not administrator accounts.
- Keep AI daily caps low until actual usage is understood.
- Download an encrypted backup every day during the trial.
- Do not upload medical records or highly sensitive images during the first trial.
- Review open community reports daily during the beta and keep the moderation inbox monitored.
- Confirm the public legal pages show the correct operator and contact details.
