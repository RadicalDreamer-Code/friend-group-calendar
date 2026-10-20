# Friend Group Calendar

Shared calendars for a friend group. Create a calendar, invite friends, add
whole-day entries with a name, date and color, and everyone can accept or
decline. Sign-in is passwordless: enter your email, type the code you get.

Web first, built mobile-first; iOS and Android come from the same Expo codebase later.

## Run it

You only need Docker.

```bash
make up
```

| Service | URL |
|---|---|
| Web app | http://localhost:8082 |
| API | http://localhost:8080/healthz |
| Mailpit (local inbox for sign-in emails) | http://localhost:8025 |
| Object storage console (RustFS, avatars) | http://localhost:9001/rustfs/console/, `calendar` / `calendar-dev-secret` |
| Postgres | `localhost:5433`, user/password/db `calendar` |

To sign in locally, enter any address on the web app and open the code from Mailpit.

Code changes reload automatically: the API is rebuilt by [air](https://github.com/air-verse/air),
the web app by Metro. Ports and credentials can be changed in a `.env` file (see `.env.example`).

Other commands (`make help` lists them all):

```bash
make logs        # follow logs
make down        # stop (data is kept)
make test        # go vet + go test (incl. database tests) in the api container
make typecheck   # TypeScript check in the web container
make gen-api     # regenerate the typed API client after editing the OpenAPI spec
make gen-sql     # regenerate the Go database code after editing SQL queries
make psql        # SQL shell
make reset-db    # delete all local data
```

### On a phone

The stack only listens on `localhost` by default. Set `HOST_BIND=0.0.0.0` in `.env`
(plus `API_PUBLIC_URL` and `API_ALLOWED_ORIGINS`, see `.env.example`) and
run `make up` so the phone can reach the API, then run the Expo dev server on your machine
instead of in Docker and open it in Expo Go or a development build:

```bash
cd apps/client
npm install
EXPO_PUBLIC_API_URL=http://<your-computer-LAN-IP>:8080 npx expo start
```

## Layout

```
apps/client      Expo + Expo Router app (web, iOS, Android), TypeScript
  src/app        routes (every file is a screen)
  src/theme      design tokens and the light/dark theme
  src/components shared UI
  src/api        typed API client generated from the OpenAPI spec
  src/auth       session state, sign-in flow, token storage (native)
services/api     Go API: chi, pgx, goose migrations (embedded, run on startup), sqlc
  internal/auth  email codes, sessions          internal/users  username, avatar
packages/api     openapi.yaml, the contract between client and API
```

## Sign-in

Passwordless: `POST /auth/request-code` emails a 6-digit code (10 minutes, 5 attempts,
stored as an HMAC), `POST /auth/verify` creates the account on first use and starts a
session. Web gets an HttpOnly, SameSite=Lax cookie; iOS and Android get a bearer token kept
in secure storage. Sessions last 90 days and renew while in use. Cross-origin writes from
browsers are rejected (Go's `CrossOriginProtection`). Rate limits are kept in memory per
email and IP. Avatars are cropped to 512 px WebP, stored in S3-compatible storage and
served by the API.

Production needs its own `AUTH_SECRET` (`openssl rand -hex 32`), an SMTP provider
(`SMTP_*`, `MAIL_FROM`; Resend and Postmark both offer SMTP) and a bucket (`S3_*`).

## Design

Minimal, Apple HIG-inspired: whitespace, hairlines, the system font and one accent.

- Accent "Fjord": `#2E78A6` light / `#63AEDB` dark, on warm neutrals (`#FAF9F7` / `#1A1918`)
- Theme: System, Light or Dark, saved per device; on web the switch crossfades via the View Transitions API
- Navigation: bottom tab bar on phones, sidebar from 768px
- All values live in `apps/client/src/theme/tokens.ts`

## Roadmap

Tracked on the Trello board "Event Planner".

1. Foundation: repo, Docker, design tokens, theme, navigation ✅
2. Email-code sign-in, long sessions, profile with username and avatar ✅
3. Friends: requests and list
4. Calendars, members, roles and invite rights
5. Entries, month view, accept / decline
6. Polish, PWA, CI, production deployment
