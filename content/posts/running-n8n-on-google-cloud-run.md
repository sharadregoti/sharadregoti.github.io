---
title: 'Problems faced while running N8N on Google Cloud Run'
date: 2025-11-01T01:15:08+05:30
draft: false
cover:
  image: "/images/running-n8n-on-google-cloud-run/cover.png"
  relative: false
---

> TL;DR: I tried running n8n on Google [Cloud Run](https://cloud.google.com/run?hl=en) without a managed database because I thought it would be cheaper. It “worked,” but it wasn’t straightforward: I had to bolt on automation for import/export state, fight Cloud Run restarts, keep CPU always allocated to keep a background backup loop alive, and deal with re-auth every few minutes when instances restarted. Fun experiment, not a recommendation.

In this post, I’ll dive deep into how I made it work, what breaks, and why I’d pick PostgreSQL for n8n in anything beyond a toy project.

All the code is available in the [GitHub repo](https://github.com/sharadregoti/deploy-n8n-google-cloud-run-sqlite-with-backups).

## Why Cloud Run for n8n?

What I know about Cloud Run:
- Serverless pricing and scaling
- Managed HTTPS and public endpoint
- Simple container deployment

I also thought Cloud Run would be cheaper, but because of a wrong calculation, I'd be under the free tier limits even if I am running 24/7. But after deploying it, I found it costs me around $60 per month, which is way more than I expected. So, in hindsight, if you are looking for a cheap n8n deployment, Cloud Run is not the right choice. You can try alternatives, such as deploying it on a small VM or using n8n Cloud itself.

### The challenge: stateless Cloud Run vs stateful n8n

How do you run a stateful application like n8n on a stateless platform like Cloud Run?

Cloud Run is stateless. Its filesystem is ephemeral, and instances can start/stop at any time. n8n with SQLite requires a writable disk that persists long enough to retain state. That mismatch is the core challenge.

## The approach: stateless n8n + GCS-backed import/export

The repo you’re reading contains a custom [image](https://github.com/sharadregoti/deploy-n8n-google-cloud-run-sqlite-with-backups/blob/master/Dockerfile) based on `n8nio/n8n:1.115.3` with two small shell scripts:

- [entrypoint.sh](https://github.com/sharadregoti/deploy-n8n-google-cloud-run-sqlite-with-backups/blob/master/entrypoint.sh) — on startup, configure rclone, pull the latest exports from a GCS bucket (workflows + credentials), import them via the n8n CLI, start a background backup loop, then execute n8n.
- [backup.sh](https://github.com/sharadregoti/deploy-n8n-google-cloud-run-sqlite-with-backups/blob/master/backup.sh) — every N seconds (default hourly), export all workflows and credentials and push them back to GCS.

That’s it: instead of persisting a database, we rehydrate on boot and periodically snapshot state to object storage.

### What’s inside the container

- Base image: `n8nio/n8n:1.115.3`
- Adds rclone v1.71.1 (as gsutils did not work with the base image)
- Entrypoint: `/bin/sh /entrypoint.sh`
- Starts n8n with `n8n start --tunnel`

### Authentication to GCS

Two options are supported out of the box:

- Service account key file mounted into the container and pointed to via `GOOGLE_APPLICATION_CREDENTIALS`
- Cloud Run Workload Identity (ADC), with the runtime service account granted `storage.objects.*` on your bucket

`RCLONE_REMOTE` names the rclone remote (e.g., `n8n-gcs`). The scripts will create/configure it on boot.

### Detailed lifecycle

1) Cold start
- Cloud Run starts a new instance
- `entrypoint.sh` configures rclone (service account or ADC)
- Attempts to download `latest/workflows.json` and `latest/credentials.json` from `gs://$BACKUP_BUCKET/$BACKUP_PREFIX/latest/`
- Imports both via `n8n import:workflow` and `n8n import:credentials`
- Spawns the backup loop and execs `n8n start --tunnel`

2) Steady state
- The backup loop sleeps and wakes per `BACKUP_INTERVAL_SEC`
- Exports to `/tmp/n8n-backup` and uploads to `latest/` in your bucket
- By default, only a `latest/` pointer is maintained for simplicity

3) Restart
- Cloud Run may restart your instance after a few minutes or hours (or when scaling)
- On boot, the process repeats, and you’re back to the last exported snapshot


**Note:** Regarding import/export of credentials:

n8n encrypts credentials using `N8N_ENCRYPTION_KEY`. If this isn’t stable across restarts, you’ll end up importing credentials that can’t be decrypted and have to sign in and reset things. So, I added this as an environment variable in the Cloud Run service.

### Backup layout and retention

By default, the script only updates:

```
$BACKUP_PREFIX/latest/workflows.json
$BACKUP_PREFIX/latest/credentials.json
```

There is code in `backup.sh` for timestamped snapshots, but it’s commented out. Enabling it will give you timestamped backups.

### Cloud Run settings that mattered

- Min instances: 1 — keeps one instance warm
- Max instances: 1 — avoids multiple writers and odd races
- CPU allocation: Always — required for cron-like loops
- Port: 5678

## The Problems I ran into

### Choosing Cloud Run Resource Allocation

Cloud Run allows you to choose between request-based and instance-based CPU allocation. I initially chose request-based allocation, thinking it would be more cost-effective. However, when I tried it out, the n8n UI didn't even open on the browser. I still didn't figure out why that happened, but after switching to instance-based allocation (one instance always running), everything started working fine. I was able to access the N8N UI on the browser. Create the account, and everything worked as expected. I was able to view my workflows and create new ones. And backup worked as expected as well.

- Keep `min instances` at 1 to reduce cold starts
- Set CPU allocation to “always” so the backup loop runs when idle (otherwise it rarely fires)

### Cloud Run restarts

Even with a warm instance, Cloud Run can and will recycle instances. With SQLite on an ephemeral disk, when the instance dies, you lose any recent changes that were not baked into the database. And the restarts were totally random. Sometimes it worked smoothly for hours, sometimes I had to re-login every few minutes.

I didn't figure out any mitigation for lost changes; if the instance restarted, I lost recent changes.

So whenever the instance restarts, I have to create the account again. I tried disabling the basic auth, but that didn't help either. I was still forced to re-create the account every few minutes. Enable basic auth (`N8N_BASIC_AUTH_ACTIVE=true`, user/password) so the editor isn’t world-open between restarts.

## Configuration and environment

Core env vars used by the scripts:

- `RCLONE_REMOTE` — e.g. `n8n-gcs`
- `BACKUP_BUCKET` — your bucket name
- `BACKUP_PREFIX` — path/prefix inside the bucket (default: `backups`)
- `BACKUP_INTERVAL_SEC` — export frequency in seconds (default `3600`)
- `N8N_ENCRYPTION_KEY` — keep this constant across deploys
- `N8N_BASIC_AUTH_ACTIVE`, `N8N_BASIC_AUTH_USER`, `N8N_BASIC_AUTH_PASSWORD`
- Optional: `GOOGLE_APPLICATION_CREDENTIALS`, `GOOGLE_CLOUD_PROJECT`

## Closing Thoughts

This was a fun exercise in bending a stateless platform to run a stateful app. The restore-on-boot trick works, but it’s fragile and operationally noisy. If you want peace of mind, give n8n the PostgreSQL it expects and self-host it on a VM, don't run it on Cloud Run.

And at the end, I hosted it on [Hertzner](https://www.hetzner.com/).

That’s it for this blog post. If you liked this post, you can subscribe to my [newsletter](https://sharad-regoti.kit.com/4efd99fc31) to stay updated. You can also check out my [YouTube channel](https://www.youtube.com/@techwithsharad), where I discuss DevOps, Cloud, Kubernetes, and AI.

If you have any questions, you can reach me on Twitter at [@SharadRegoti](https://twitter.com/SharadRegoti)