# FormForge 1.4 Backup and Recovery

## What backups contain

Version 1.4 encrypted backups contain the database plus encrypted progress-photo files. They exclude active session tokens and login-attempt state. The outer `.ffbackup` is encrypted with the installation recovery key and integrity checked before being accepted.

## Automatic jobs

FormForge records automatic backup and integrity-check jobs. The default backup interval is 24 hours and can be changed by an administrator. The health endpoint becomes degraded when backups are overdue or jobs have failed. Rotation keeps recent daily copies and representative weekly copies. An optional copy folder may point to OneDrive or another selected local/synced folder.

## Manual backup

Open **Settings → Encrypted backups → Back up now**, then download the resulting `.ffbackup`. Export the recovery key separately from **Settings**. Do not store the only copy of the recovery key beside the only backup.

## Restore

1. Create a new manual backup of the current installation.
2. Choose the `.ffbackup` under **Restore encrypted backup**.
3. Enter the recovery key if the backup came from another installation.
4. FormForge decrypts and validates the archive, runs required schema migrations, restores photo files safely, clears all sessions, and requires a restart/sign-in.
5. Migration failures do not overwrite the active database.

## Corruption recovery

FormForge writes atomically and keeps `formforge.db.json.prev`. During startup it can recover a valid previous copy when the primary database is damaged. Before a schema migration it also saves a safety copy under `migration-backups`. Do not manually edit the database unless you have backups and understand the schema.

## Recovery-key loss

A backup cannot be decrypted without the correct recovery key. The same key can also reset an administrator password, so treat it as a high-value secret. Rotating the recovery key affects future backups; keep old keys for old backups until those backups are retired.
