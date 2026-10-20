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
| Postgres | `localhost:5433`, user/password/db `calendar` |

Code changes reload automatically: the API is rebuilt by [air](https://github.com/air-verse/air),
the web app by Metro. Ports and credentials can be changed in a `.env` file (see `.env.example`).

Other commands (`make help` lists them all):

```bash
make logs        # follow logs
make down        # stop (data is kept)
make test        # go vet + go test in the api container
make typecheck   # TypeScript check in the web container
make gen-api     # regenerate the typed API client after editing the OpenAPI spec
make psql        # SQL shell
make reset-db    # delete all local data
```

### On a phone

The stack only listens on `localhost` by default. Set `HOST_BIND=0.0.0.0` in `.env` and
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
services/api     Go API: chi, pgx, goose migrations (embedded, run on startup)
packages/api     openapi.yaml, the contract between client and API
```

## Design

Minimal, Apple HIG-inspired: whitespace, hairlines, the system font and one accent.

- Accent "Fjord": `#2E78A6` light / `#63AEDB` dark, on warm neutrals (`#FAF9F7` / `#1A1918`)
- Theme: System, Light or Dark, saved per device; on web the switch crossfades via the View Transitions API
- Navigation: bottom tab bar on phones, sidebar from 768px
- All values live in `apps/client/src/theme/tokens.ts`

## Roadmap

Tracked on the Trello board "Event Planner".

1. Foundation: repo, Docker, design tokens, theme, navigation ✅
2. Email-code sign-in, long sessions, profile with username and avatar
3. Friends: requests and list
4. Calendars, members, roles and invite rights
5. Entries, month view, accept / decline
6. Polish, PWA, CI, production deployment
