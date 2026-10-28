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
make lint        # ESLint, Prettier and the color contrast check for the client
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

CI (GitHub Actions) runs the same checks on every push and pull request, plus `gofmt`, the
race detector and a build of both production images.

## Production

`deploy/` has everything for one server: a Compose file with Caddy (automatic HTTPS, serving
the web app at `app.DOMAIN` and proxying `api.DOMAIN`), the API, Postgres and a daily
`pg_dump`. Setup, updates and restoring a backup are in [deploy/README.md](deploy/README.md).

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
  internal/friends  friend requests, friendships, username search
  internal/calendars  calendars, members, roles, entries, answers and who may do what
packages/api     openapi.yaml, the contract between client and API
deploy           production Compose file, Caddyfile, backup script
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
(`SMTP_*`, `MAIL_FROM`; Resend and Postmark both offer SMTP) and a bucket (`S3_*`);
`deploy/.env.example` explains each setting.

## Friends

Find people by username (prefix search, at most 20 results, never showing email addresses),
send a request, and they accept or decline. Asking someone who already asked you makes you
friends straight away. Each pair of users has at most one row in `friendships`, pending or
accepted; declining, cancelling and removing delete it. Requests are rate limited (30 an hour,
3 a day to the same person) so a declined request cannot be repeated endlessly. Friend features
need a username first.

## Calendars

Anyone with a username can create a calendar and becomes its owner. There is exactly one
owner, who renames or deletes the calendar, changes members' roles, removes members and can
make another member the owner (and then stays on as an editor who can invite). Editors add,
edit and delete entries, viewers see them and accept or decline. The owner can invite friends, and so
can members who were given the invite right, either one by one or with "Everyone can invite".
Only your own friends can be invited; they join right away, as viewers unless you pick a role,
and never with a higher role than yours. Everyone but the owner can leave.

All permissions are decided in one place, `Membership.Can` in `internal/calendars`, and every
change locks the calendar row first so concurrent changes (two transfers at once, say) cannot
leave it in a broken state. People who are not members get `404 calendar_not_found`, so
calendar IDs reveal nothing. Limits: 50 members per calendar, 50 owned calendars per user.

## Entries

Entries are whole days with a name, a date (`YYYY-MM-DD`, no time zone) and one of eight preset
colors. The owner and editors add, edit and delete them, and every member accepts or declines;
not answering is pending, and an answer can be taken back. Moving an entry to another day clears
its answers, because they were for the old day. Leaving a calendar drops your answers in it.
Entries are listed a month at a time (`GET /calendars/{id}/entries?month=2026-10`), with the IDs
of who accepted and who declined; the client matches them against the members it already has.
Limits: 20 entries a day, 5000 per calendar, dates from 2000 to 2099.

The client keeps shared data fresh without live updates: queries refetch when the app or tab
comes back into focus or a screen is shown again, and calendar screens poll every 30 seconds
while they are on screen. Answers show right away and are sent in order, so the last tap wins.

## Use with Claude

The API has an MCP endpoint at `/mcp`, so Claude Code (or any other MCP client) can look at
your calendars and add entries for you. Create a personal access token under Profile → Connect
an AI assistant, then add the server:

```sh
claude mcp add --transport http friend-cal http://localhost:8080/mcp --header "Authorization: Bearer fgc_…"
```

Any other MCP client takes the URL (`http://localhost:8080/mcp` locally, `https://api.<domain>/mcp`
in production) and the header `Authorization: Bearer fgc_…`. The transport is Streamable HTTP,
stateless, with plain JSON responses.

Tools: `list_calendars`, `list_entries` (a calendar's entries between two dates, at most 92 days,
with who is going) and `create_entry` (owners and editors; at most 30 an hour). Everything goes
through the same permission checks as the app, and a viewer gets a tool error rather than a new
entry.

Tokens are `fgc_` plus 32 random bytes, stored as a SHA-256 hash and shown once. They expire
after 30, 90 or 365 days, can be revoked on the same screen, and only `/mcp` accepts them: the
REST API keeps taking sessions only, so a token cannot sign in to the app or create more tokens.
At most 10 live tokens per user.

## Design

Minimal, Apple HIG-inspired: whitespace, hairlines, the system font and one accent.

- Accent "Fjord": `#2E78A6` light / `#63AEDB` dark, on warm neutrals (`#FAF9F7` / `#1A1918`)
- Theme: System, Light or Dark, saved per device; on web the switch crossfades via the View Transitions API
- Navigation: bottom tab bar on phones, sidebar from 768px
- All values live in `apps/client/src/theme/tokens.ts`; colors in `colors.ts`, where every
  text and background pair has to pass WCAG AA (`npm run check:contrast`)
- Loading shows skeleton rows, failures a way to try again; data loaded before stays on
  screen when a refresh fails
- Motion is short and calm, and off for people who prefer reduced motion; haptics on iOS and Android
- Installable as a web app (manifest and icons in `apps/client/public`; icon sources in
  `apps/client/assets/source`, rendered by `assets/source/render.sh`)

## Roadmap

Tracked on the Trello board "Event Planner".

1. Foundation: repo, Docker, design tokens, theme, navigation ✅
2. Email-code sign-in, long sessions, profile with username and avatar ✅
3. Friends: requests and list ✅
4. Calendars, members, roles and invite rights ✅
5. Entries, month view, accept / decline ✅
6. Polish, PWA, CI, production deployment ✅
