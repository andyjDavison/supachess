# Supachess: Project Plan

Chess.com-style app: users register, log in, get matched, play real-time games, and later friend and message each other.
This plan was written from design conversations, not from the code. If anything here disagrees with the actual code, trust the code and update this file.

## Status

### Done

- [x] Auth: register, login, logout, `/api/me`, `RequireAuth`, frontend `ProtectedLayout`, login modal with a pending action
- [x] Game domain, Postgres repository, move validator, `GameService`, game-expiry timers
- [x] WebSocket hub, dispatcher, event publisher, origin check
- [x] Matchmaking with 9 presets (1+0, 1+1, 2+1, 3+0, 3+2, 5+0, 10+0, 10+5, 15+10)
- [x] Game page: board, legal-move dots, capture rings, right-click marks, castling, orientation, live clocks, game-over modal
- [x] Token-based restyle of navbar, play page, and play-online page

### Next (in order)

- [x] **Move list:** render `GET /api/games/{id}/moves` in the `SidePanel` placeholder, appending live from `game.move_made`
- [x] **Finish game page layout:** board plus side panel in one flex row (see "Layout notes"). Restyle `GameBoard`, `GameOverModal`, and the auth forms with the token system.
- [x] **Backfill Tests for Frontend & Backend:** most of the codebase is missing test coverage/good testing, and needs to be tested before deploying to prod so we have solid insurance on deployments.
- [x] **First deploy:** AWS Lightsail VM, Docker Compose (postgres, api), TLS and reverse proxy, frontend on Vercel, prod env vars. Do this before friends and chat, because it surfaces real problems early.
- [ ] **Friends:** new `internal/friends` domain (service, repository, HTTP handlers, migration), plus a `features/friends` frontend. Needs friend requests, a friends list, and blocking.
- [ ] **Play a Friend:** add a `waiting` game status and an invite flow via WS (offer, accept, decline).
- [ ] **Chat:** reuse the per-user WebSocket with new message types for DMs and in-game chat. Persist messages. Design this before building.
- [ ] **Final Styling:** make style changes/fixes as we go, at the end we should be able to have styles cleaned up and fit many different screens

### Known gaps (deliberately deferred)

- [ ] Threefold repetition: `ValidateMove` receives `history` but ignores it
- [ ] `AppendMove` and `Update` should share one DB transaction (consider a combined `RecordMove` repo method)
- [ ] Timers are lost on restart: rehydrate from active games on startup
- [ ] `SubmitMove` read-modify-write race: needs a per-game lock or optimistic versioning
- [ ] Matchmaking always makes the first player white: balance by color history
- [ ] Draw offers: decide whether they ship in v1. `offer_draw` and `respond_draw` are placeholders.
- [ ] Promotion always picks a queen: add a picker UI
- [ ] "Analyze game" button in `GameOverModal` is a no-op
- [ ] `Hub` holds one connection per user, so a second tab or device replaces the first
- [ ] Matchmaking queue is in-memory, so it won't work across multiple server instances (use Redis if scaling)

### Cleanup and housekeeping

- [ ] Register validation errors return 500: `validateRegisterInput` uses plain `errors.New`, so `writeServiceError` can't map them. Add a `domain.ErrInvalidInput` sentinel → 400 (plus its WS wire code and protocol-doc row)
- [ ] Add to the protocol doc: error codes `invalid_message`, `internal_error`, `unknown_preset`, `already_in_game`
- [ ] Confirm registration maps the ELO bracket to a starting rating (it may still hardcode 1200)
- [ ] Restore `domain.UserID` and `domain.GameID` types in the game code (it currently uses plain `string`)
- [ ] Audit every handler for rule 1
- [ ] Make CORS and the WS origin check read from one config value
- [ ] Remove `console.log` leftovers
- [ ] Optional: tidy the Vite HMR websocket warning and `npm audit` advisories

## Layout notes

- The game page should be one flex row filling the viewport: board column (flex 1, `min-width: 0`) next to a fixed-width side panel.
- The board is a square. Size its wrapper with `min(100%, calc(100cqh - <rows and gaps>))` inside a `container-type: size` parent, so it shrinks with either width or height.
- The `Chessboard` component sizes itself from its container's width, so fix sizing with a wrapper, not a prop.

## Commands

- `make run` / `make test` / `make lint` / `make tidy`
- `make docker-up` / `make docker-down`
- `make migrate-up` / `make migrate-down` / `make migrate-version` / `make migrate-create name=<name>`
- Run everything from the repo root. `.env` must exist (copy it from `.env.example`).

## Working agreements

- Update the checkboxes in this file as work lands.
- Keep changes small and tested. Add unit tests for services, using the in-memory repositories.
- Ask before changing a wire format, a migration that has already run, or the dependency direction described above.
