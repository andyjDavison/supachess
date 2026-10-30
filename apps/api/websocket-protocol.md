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
| `chat.send_direct`    | `toUserId`, `body`                   | DM anyone who isn't blocked either way (a block answers `user_not_found`). Creates the conversation on first contact. `body` is trimmed and must be 1–500 characters. Success is the `chat.direct_message` echo, not a separate ack. |
| `chat.send_game`      | `gameId`, `body`                     | Players only, and only while the game is `active`. Success is the `chat.game_message` echo.                                                                                                                       |

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
| `chat.direct_message`        | `conversation` (Conversation), `message` (DirectMessage)            | To **both** members, each with their own view of `conversation` (`folder`, `unread`, `other`) — the sender's copy is how their own other views stay in step. Full current state, not a delta.                                                                        |
| `chat.game_message`          | `gameId`, `message` (GameMessage)                                   | To both players.                                                                                                                                                                                                                                                    |
| `error`                      | `code`, `message`, `inReplyTo?`                                                   | Generic envelope for anything off the happy path: illegal move, not-your-turn, resigning an already-finished game, etc. `code` is the machine-readable part the frontend actually branches on (see below); `message` is for logging/fallback display, not UI logic. |

## Chat shapes

Reading chat is REST (`GET /api/conversations`,
`/api/conversations/{id}/messages?before=<id>`, `POST …/read`,
`PUT|DELETE …/archive`, `GET /api/games/{id}/chat`); only sending is on the
socket. Both use these shapes:

- **DirectMessage** — `id` (number, increasing; the paging cursor),
  `conversationId`, `senderId`, `body`, `sentAt` (unix ms).
- **Conversation** — `id`, `other` (`userId`, `username`, `rating`),
  `folder`, `unread` (boolean), `lastMessage` (DirectMessage or `null`).
  Always one member's view.
  - `folder` is `inbox` (you've sent a message in it), `unread` (the other
    member started it and you haven't replied), or `archived` (you archived
    it). A new message from either side un-archives it for both.
  - `unread` means there's a message from the other member you haven't
    opened since; independent of `folder`.
- **GameMessage** — `id`, `gameId`, `senderId`, `body`, `sentAt` (unix ms).

Sends are rate limited per user, shared across DMs and game chat: a burst
of 5, then one per second (`rate_limited`).

## Error envelope

`inReplyTo` is the `type` of the client message that caused the error,
so a client can route it to whatever sent the request — `user_not_found`
means something different to an `invite.offer` than to a
`chat.send_direct`. It's omitted only when the message was too malformed
to have a readable `type`.

## Game fields

Every `Game` payload (REST and WebSocket) carries: `id`, `whiteId`,
`blackId`, `fen`, `status`, `result?`, `resultReason?`,
`whiteTimeRemainingMs`, `blackTimeRemainingMs`, `initialMs`,
`incrementMs`, `lastMoveAt`, `finishedAt?`, `ratings?`. `initialMs` and
`incrementMs` are the time control (e.g. 3+2 is `180000` / `2000`); all
durations are milliseconds and timestamps unix millis.

`ratings` is omitted only for games from before ratings existed:

```json
"ratings": {
  "category": "bullet",
  "white": { "rating": 1200, "win": 18, "draw": 8, "loss": -2, "delta": 18 },
  "black": { "rating": 1600, "win": 2, "draw": -8, "loss": -18, "delta": -18 }
}
```

- `category` is `bullet`, `blitz` or `rapid`, from the time control
  (initial + 40 × increment: under 3 min Bullet, under 10 Blitz, else
  Rapid). Each player has a rating per category.
- `rating` is each player's rating in that category when the game was
  created; `win` / `draw` / `loss` are what each result would change it
  by (Elo, K=40 for a player's first 30 games in the category, then 20).
- `delta` appears once the game is finished: the change actually applied.

The `rating` on `matchmaking.matched` and `invite.*` opponents is also the
rating for the game's category. Everywhere else a single rating is shown
(friends, chat, `/api/me`) it's the Blitz rating.

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
| `conversation_not_found`   | `domain.ErrConversationNotFound`  |
| `invalid_message_body`     | `domain.ErrInvalidMessageBody`    |
| `cannot_message_self`      | `domain.ErrCannotMessageSelf`     |
| `rate_limited`             | `domain.ErrRateLimited`           |
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
