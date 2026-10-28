# WebSocket message protocol

This is the source of truth for every message that flows over the
WebSocket connection, in both directions. The Go backend's message
structs and the frontend's Zod schemas should both be written _from_ this
document, and both updated together whenever it changes — this file has
no compiler enforcing it stays in sync with either side, so treat it as
the contract and review it alongside any change to either implementation.

## Connection model

- **One connection per logged-in user**, not per game. A user opens a
  single WebSocket on login (or app load, if a valid session already
  exists) and keeps it open across matchmaking, active games, chat, and
  friend presence — not reconnecting per game.
- **Addressed by user ID.** The server's connection registry (the `Hub`)
  is keyed by user ID, not game ID. A message concerning a specific game
  carries a `gameId` field in its payload; it's not routing based on
  which "room" the socket belongs to.
- **Auth** happens at connect time — the token is passed as a query
  param on the upgrade request (`/ws?token=...`) and verified before the
  connection is registered. There is no per-message auth; the connection
  itself is the authenticated session.
- **`subscribe_game`** is how a client tells the server which game(s) it
  wants game-scoped messages for — sent right after receiving `matched`,
  or on reconnect/page-load when resuming a game already in progress.

## Envelope

Every message, both directions, uses the same outer shape:

```json
{
  "type": "game.move_made",
  "payload": { ... }
}
```

- `type` is a scoped string discriminator: `<domain>.<event>`. Scoped
  rather than flat (`move_made` vs. `game.move_made`) because one
  connection carries messages from multiple domains — game, matchmaking,
  chat, presence — and scoping avoids name collisions as those are added.
- Receivers branch on `type` first, then validate/parse `payload`
  against the schema that type implies.
- Every payload identifying a specific game includes `gameId` explicitly
  — never inferred from connection state, since one connection can be
  involved with things outside any game context (matchmaking, presence).

## Client → Server

| `type`                | Payload                              | Notes                                                                                                                                                                                                              |
| --------------------- | ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `game.submit_move`    | `gameId`, `from`, `to`, `promotion?` | `promotion` omitted unless the move is a pawn reaching the back rank. **No `playerId` field** — the server identifies the sender from the authenticated connection itself, never from client-claimed payload data. |
| `game.resign`         | `gameId`                             |                                                                                                                                                                                                                    |
| `game.offer_draw`     | `gameId`                             | Status: **not yet decided** whether draw-by-agreement ships in v1 — placeholder, confirm before implementing either side.                                                                                          |
| `game.respond_draw`   | `gameId`, `accept: boolean`          | Same open-decision caveat as above.                                                                                                                                                                                |
| `game.subscribe_game` | `gameId`                             | Sent after `matchmaking.matched`, or on reconnect to resume an in-progress game.                                                                                                                                   |
| `matchmaking.join`    | `timeControl?`                       | Payload empty/omitted if only one fixed time control is supported for now.                                                                                                                                         |
| `matchmaking.leave`   | _(none)_                             | Explicit cancel; server already knows who's asking from the connection.                                                                                                                                            |
| `invite.offer`        | `toUserId`, `preset`                 | Challenge a specific player. `preset` is one of the matchmaking presets (`"3+2"`, …); colors are a server-side coin flip. Online-only: fails with `user_not_connected` if the invitee has no open socket. One outgoing invite per user. |
| `invite.accept`       | `gameId`                             | Invitee only. Starts the game; both players then get `invite.accepted`.                                                                                                                                            |
| `invite.decline`      | `gameId`                             | Invitee only.                                                                                                                                                                                                      |
| `invite.cancel`       | `gameId`                             | Inviter only.                                                                                                                                                                                                      |

## Server → Client

| `type`                       | Payload                                                             | Notes                                                                                                                                                                                                                                                               |
| ---------------------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `game.move_made`             | `game` (full current `Game` state), `move` (the `Move` just played) | Sends **full current state**, not a delta — a client that missed a prior message (brief disconnect) can resync from the very next message it receives, no separate "give me full state" round trip needed.                                                          |
| `game.game_over`             | `game` (full state, including `result`, `resultReason`)             |                                                                                                                                                                                                                                                                     |
| `game.draw_offered`          | `gameId`, `offeredBy`                                               | Contingent on the same open draw-by-agreement decision above.                                                                                                                                                                                                       |
| `game.opponent_disconnected` | `gameId`                                                            | Distinct from `game_over` — the game isn't over, the opponent's connection just dropped.                                                                                                                                                                            |
| `game.opponent_reconnected`  | `gameId`                                                            |                                                                                                                                                                                                                                                                     |
| `matchmaking.matched`        | `gameId`, `opponent` (username, rating)                             | Ends a client's `matchmaking.join` wait; the client's cue to navigate into the game and send `game.subscribe_game`.                                                                                                                                                 |
| `invite.sent`                | `gameId`, `to` (userId, username, rating), `preset`, `expiresAt`    | To the inviter: the offer was created. `gameId` is the invite's handle for `invite.cancel`. `expiresAt` is unix millis (60s after the offer).                                                                                                                        |
| `invite.received`            | `gameId`, `from` (userId, username, rating), `preset`, `expiresAt`  | To the invitee.                                                                                                                                                                                                                                                     |
| `invite.accepted`            | `gameId`, `opponent` (username, rating)                             | To both players. Same shape as `matchmaking.matched` and handled the same way: navigate into the game and send `game.subscribe_game`.                                                                                                                               |
| `invite.closed`              | `gameId`, `reason`                                                  | To both players when an invite ends without a game. `reason` is `declined`, `cancelled`, `expired`, or `unavailable` (a party disconnected, or one is now in another game).                                                                                          |
| `error`                      | `code`, `message`                                                   | Generic envelope for anything off the happy path: illegal move, not-your-turn, resigning an already-finished game, etc. `code` is the machine-readable part the frontend actually branches on (see below); `message` is for logging/fallback display, not UI logic. |

## Game status

The `status` on every `Game` payload is one of:

- `waiting` — an `invite.offer` has created the game but it hasn't been
  accepted. No clock runs. If it's declined, cancelled or expires, the
  game is deleted rather than finished.
- `active` — in play. Clocks start when a game becomes active.
- `finished` — `result` and `resultReason` are set.

## Error codes

Mirror the backend's `domain.Err*` sentinels directly — one-to-one,
so adding a new domain error and forgetting to add its wire-level code
is easy to catch by just diffing this list against `domain/errors.go`.

| `code`                  | Corresponds to                  |
| ----------------------- | ------------------------------- |
| `illegal_move`          | `domain.ErrIllegalMove`         |
| `not_your_turn`         | `domain.ErrNotYourTurn`         |
| `game_not_found`        | `domain.ErrGameNotFound`        |
| `game_already_finished` | `domain.ErrGameAlreadyFinished` |
| `cannot_friend_self`       | `domain.ErrCannotFriendSelf`      |
| `already_friends`          | `domain.ErrAlreadyFriends`        |
| `friend_request_exists`    | `domain.ErrFriendRequestExists`   |
| `friend_request_not_found` | `domain.ErrFriendRequestNotFound` |
| `not_friends`              | `domain.ErrNotFriends`            |
| `invite_not_found`         | `domain.ErrInviteNotFound`        |
| `invite_already_pending`   | `domain.ErrInviteAlreadyPending`  |
| `cannot_invite_self`       | `domain.ErrCannotInviteSelf`      |
| `user_not_found`           | `domain.ErrUserNotFound`          |
| `user_not_connected`       | `domain.ErrUserNotConnected`      |
| `unknown_preset`           | `domain.ErrUnknownPreset`         |
| `already_in_game`          | `domain.ErrAlreadyInGame`         |
| `invalid_message`          | _(none)_ — the envelope or payload didn't parse |
| `internal_error`           | _(none)_ — any unmapped error; details are logged server-side, not sent |

The friend codes are reserved for when friend events move onto the
socket; today the friends feature is REST-only (`/api/friends/*`) and
these sentinels surface as HTTP statuses instead.

`user_not_found` is also what an `invite.offer` gets when either player
has blocked the other — deliberately the same as for a user that doesn't
exist.

## Known open decisions

Tracked here rather than silently assumed — resolve before building the
piece that depends on the answer:

- **Draw-by-agreement (v1 or later?)** — affects `game.offer_draw`,
  `game.respond_draw`, `game.draw_offered`, and the `DrawOfferedBy` field
  on `domain.Game`.
- **Time control selection** — fixed default for now, or client-chosen
  at `matchmaking.join` time? Affects whether matchmaking needs separate
  sub-queues per time control.

## Changelog

- _Not yet versioned — add entries here once the protocol is in active
  use and changes need to be tracked against what either side has
  actually implemented._
