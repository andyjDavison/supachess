# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.
Read PLAN.md before starting any task. It holds the prioritized task list. Update its checkboxes when you finish something.
I will give you one checkbox at a time, enter plan mode first so I can review your approach before you edit something.

## Layout

A chess website. Two apps under `apps/`, plus SQL migrations — no monorepo tooling, just a `Makefile` at `apps/` that shells into each.

- `apps/api` — Go 1.26 HTTP + WebSocket server (module name is just `api`, so imports read `api/internal/...`)
- `apps/ui` — React 19 + Vite + Tailwind v4 SPA, pnpm
- `apps/db/migrations` — golang-migrate numbered up/down pairs
- `apps/api/websocket-protocol.md` — hand-maintained contract for every WS message; see below

## Commands

All from `apps/`:

```
make docker-up            # Postgres 16 on :5432 (postgres/admin/postgres)
make run-server           # api on :8080; applies migrations on boot
make run-frontend         # vite dev server on :5173
make test-server          # cd api && go test ./...
make test-frontend        # cd ui && pnpm test   (vitest, watch mode)
make build-server / make build-frontend
make migrate-create name=add_foo_column
make migrate-up / migrate-down / migrate-version / migrate-force version=N
```

`make lint` is broken — it runs `go vet ./...` from `apps/`, which has no `go.mod`. Use `cd api && go vet ./...`. The frontend linter is `cd ui && pnpm lint` (not in the Makefile).

A single vitest file: `cd ui && pnpm vitest run src/components/navbar/navbar.test.tsx`. A single Go test: `cd api && go test ./internal/game -run TestName`. There are currently **no Go test files at all**, and only two trivial UI tests.

Migrations also run automatically from `main.go` on every server start (`RunMigrations`), so `make migrate-up` is only needed to apply schema changes without restarting.

## Backend architecture

Layered, with interfaces owned by the inner layer:

```
transport/web (HTTP)  transport/ws (WebSocket)
                  \        /
          game.GameService, matchmaking.MatchmakingService, user, auth
                       |
             repositories (Postgres)  /  domain (pure types)
```

The important structural rule: **services never import the transport layer.** `game` declares `GameEventPublisher` and `TimerScheduler`; `matchmaking` declares `Notifier`, `gameService`, `userLookup`, `Clock`. Implementations live in `transport/ws` (`GameEventPublisher`, `MatchmakingNotifier`) and satisfy those interfaces structurally. Wiring happens in exactly one place, `cmd/main.go`. `GameService` and `TimerManager` are mutually dependent, so main.go constructs the service with a nil scheduler and calls `SetTimers` afterward.

`domain/errors.go` holds sentinel errors; every layer propagates them with `%w` and the edges map them to a status code (`web`) or a wire error code (`ws.sendServiceError`). Add a new `domain.Err*` and you must add its wire code to both `sendServiceError` and the protocol doc's error table.

Note `game.GameRepository` is a concrete struct, not an interface, despite the surrounding comments about seams — `GameService` depends on it directly.

### Authoritative server state

The server owns everything the client could lie about: move legality (`MoveValidator` wrapping `notnil/chess`), whose turn it is (parsed from the FEN, `activeColorFromFEN`), player identity (taken from the authenticated connection, never from the message payload), and time controls (clients send a preset _name_ like `"3+2"`; `matchmaking/preset.go` maps it to a duration).

Clocks are computed, not ticked: each move subtracts `now - game.LastMoveAt` and adds the increment. Between moves nothing is pushed, so the UI's `useLiveClocks` derives the display from `lastMoveAt` + wall clock. One `time.AfterFunc` per active game (`TimerManager`) fires `HandleTimeExpired`. That map is in-memory — **a server restart silently loses every running game's timer**, with no rehydration.

### Matchmaking

`Queue` partitions waiting players by preset. Pairing is attempted both on join and on a 2s sweep ticker (`RunSweepLoop`, started as a goroutine in main.go) because rating brackets widen with wait time (`bracket.go`: ±100, +50 every 15s, capped at ±400) — so a pairing can become legal purely from elapsed time with no event to trigger it. `allowedRange` takes `now` as a parameter specifically so widening is testable without sleeping. Colors are assigned arbitrarily (first entry = white).

### WebSocket layer

One connection per logged-in user, not per game — opened on login, kept across matchmaking and games. `Hub` is keyed by user ID, so every game-scoped payload carries `gameId` explicitly. Known simplification: one entry per user ID, so a second tab overwrites the first.

Each connection runs a read pump and a write pump; `gorilla/websocket` forbids concurrent writes, so `Send` only enqueues onto a 32-slot buffered channel and the single `writePump` goroutine owns the socket. A full buffer disconnects that client rather than applying backpressure.

`ws.ConfigureAllowedOrigins` must be called before serving (main.go does) — an unrestricted `CheckOrigin` is a CSRF hole once auth is cookie-based.

## Wire format

`apps/api/websocket-protocol.md` is the contract, and nothing enforces it: the Go structs in `transport/ws` + `transport/dto` and the Zod schemas in `ui/src/lib/ws/schemas.ts` are both written _from_ it by hand. **Change any of the three and change all three in the same commit.** The frontend logs loudly (never silently drops) when a message fails `serverMessageSchema` validation — that's the drift detector.

Every message is `{ type: "<domain>.<event>", payload: {...} }`. Server→client messages send **full game state**, not deltas, so a client that missed a frame resyncs from the next one.

Durations cross the wire as **milliseconds** (`dto.GameDTO.WhiteTimeRemainingMs`, timestamps as unix millis) while Go uses `time.Duration` and Postgres stores raw `int64` nanoseconds in `white_time`/`black_time`/`initial`/`increment`. Three different units for the same value — convert at the boundary.

Draw-by-agreement is unimplemented on purpose: `OfferDraw`/`RespondToDraw` return `nil, nil`, and the protocol doc lists it as an open decision.

## Frontend architecture

`main.tsx` nests `AuthProvider` → `AuthModalProvider` → `RouterProvider`. Routes are declared centrally in `src/routes/routes.ts` (react-router v8, `Component:` not `element:`), with `RootLayout` supplying the navbar chrome per section and `ProtectedLayout` guarding authenticated routes.

Code is grouped by feature under `src/features/<feature>/`, with cross-feature pieces in `src/components`, `src/lib`, and `src/stores`.

Two separate pieces of state, deliberately:

- **Auth** — React context (`stores/AuthContext.tsx`). Session is an httpOnly cookie, so on load it asks `GET /api/me` rather than reading a token. The WS connection is opened/closed as a side effect of `user` changing.
- **Game/matchmaking** — a zustand store (`features/game/store.ts`). `gameSocket` (`lib/ws/client.ts`) is a module-level singleton wrapping `partysocket` for reconnects; the store subscribes to it once at creation and is the only thing that sends.

Two auth patterns coexist on purpose: `ProtectedLayout` redirects to `/login` for whole routes, while `useRequireAuth()` wraps an action so the login modal opens and replays the queued action without navigating away.

Tailwind v4 — no config file. Theme colors live in an `@theme` block in `src/index.css` (`bg-nav-bg`, `text-text`, `bg-options-header`, …), with board layout helpers as `@layer components` classes.

## Environment and known inconsistencies

`apps/api/.env` is loaded by godotenv (cwd is `api/` under `make run-server`); `apps/.env` is included by the Makefile for the `migrate-*` targets; `apps/ui/.env` holds `VITE_API_URL` / `VITE_WS_URL`.

Live mismatches to be aware of before "fixing" something that looks wrong:

- The protocol doc says WS auth is `/ws?token=...`; the implementation actually uses the same `auth.RequireAuth` session cookie as the REST routes.

## Deploy

Frontend on Vercel (`apps/ui`, `vercel.json` SPA rewrite), API + Postgres + Caddy via `apps/deploy/docker-compose.prod.yml` on an Oracle arm64 VM, image built by `.github/workflows/api-image.yml` into GHCR. Runbook: `apps/deploy/README.md`. Frontend and API must stay on one registrable domain (`<domain>` / `api.<domain>`) or the `SameSite=Lax` session cookie stops being sent. With `IS_PRODUCTION=true` the api refuses to start without `JWT_SECRET`.
