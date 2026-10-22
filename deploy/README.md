# Production deployment

One server with Docker runs everything: Caddy (HTTPS and the web app), the API,
Postgres and a daily backup. Email goes out through an SMTP provider and avatars
live in an S3-compatible bucket.

```
browser ──► Caddy :443 ─┬─ app.DOMAIN ──► web app (static files)
                        └─ api.DOMAIN ──► API ──► Postgres ◄── backup (daily, into ./backups)
                                           ├────► SMTP provider (sign-in codes)
                                           └────► bucket (avatars)
```

## What you need

- A Linux server with Docker and the Compose plugin, ports 80 and 443 open. The images
  are built on the server; the web app's build is the heaviest step, so give a small
  machine some swap.
- A domain with two DNS records pointing at the server: `app.example.com` and
  `api.example.com` (A, and AAAA if the server has IPv6).
- An SMTP provider with a verified sender domain, e.g. [Resend](https://resend.com) or
  [Postmark](https://postmarkapp.com).
- A private bucket and an access key, e.g. Cloudflare R2 (Object Read & Write on that
  bucket only) or AWS S3.

## First start

```bash
git clone https://github.com/RadicalDreamer-Code/friend-group-calendar.git
cd friend-group-calendar/deploy
cp .env.example .env
nano .env                       # every value; openssl rand -hex 32 for the secrets
docker compose up -d --build
docker compose ps               # api and db turn "healthy" after a few seconds
```

Caddy fetches the certificates on the first request; open `https://app.example.com`
and sign in. The first sign-in email shows whether SMTP works; `docker compose logs api`
shows why if it does not.

## Updates

```bash
cd friend-group-calendar
git pull
cd deploy && docker compose up -d --build
```

Migrations run when the API starts. The web app's address for the API is built into it,
so changing `DOMAIN` needs the `--build` too.

## Backups

The `backup` service writes `pg_dump` files to `deploy/backups/` every day at
`BACKUP_HOUR_UTC` and deletes the ones older than `BACKUP_KEEP_DAYS`. They are on the
same disk as the database, so copy them somewhere else as well, for example with a cron
job running `rclone copy deploy/backups remote:calendar-backups`. Avatars are in the
bucket and not part of the dump.

```bash
docker compose exec backup sh /backup.sh now        # one backup right away
docker compose logs backup                          # what the last runs did
```

Restore a dump (this replaces the current data):

```bash
docker compose stop api
docker compose exec -T db sh -c 'pg_restore --clean --if-exists --no-owner -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < backups/calendar-2026-10-23-0300.dump
docker compose start api
```

## Good to know

- Sign-in codes and sessions are only as safe as `AUTH_SECRET` and the database: keep
  `.env` readable by root only (`chmod 600 .env`).
- The API is not published on a port; only Caddy reaches it. It records the client IP
  from Caddy's `X-Forwarded-For` (`TRUSTED_PROXIES` is Caddy's address), which the sign-in
  rate limits count by.
- Logs: `docker compose logs -f api` (JSON, one line per request) and `docker compose logs web`.
