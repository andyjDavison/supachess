import {
  gameSchema,
  moveSchema,
  movesResponseSchema,
  serverMessageSchema,
} from "./schemas";

// These schemas are written by hand from websocket-protocol.md, with
// nothing enforcing they match the Go side. The frontend's only defence
// against drift is that a mismatched message fails to parse loudly, so
// these tests pin the shapes the backend actually sends.

const validGame = {
  id: "game-1",
  whiteId: "white-1",
  blackId: "black-1",
  fen: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
  status: "active",
  whiteTimeRemainingMs: 180_000,
  blackTimeRemainingMs: 180_000,
  initialMs: 180_000,
  incrementMs: 0,
  lastMoveAt: 1_700_000_000_000,
};

const validMove = {
  gameId: "game-1",
  ply: 1,
  color: "white",
  from: "e2",
  to: "e4",
  san: "e4",
  fenAfter: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
  playedAt: 1_700_000_000_000,
};

describe("gameSchema", () => {
  it("accepts an in-progress game with result fields omitted", () => {
    const parsed = gameSchema.parse(validGame);

    expect(parsed.status).toBe("active");
    expect(parsed.result).toBeUndefined();
  });

  it("accepts a finished game with its result", () => {
    const parsed = gameSchema.parse({
      ...validGame,
      status: "finished",
      result: "white_wins",
      resultReason: "checkmate",
      finishedAt: 1_700_000_100_000,
    });

    expect(parsed.result).toBe("white_wins");
  });

  it("accepts a null finishedAt", () => {
    expect(() => gameSchema.parse({ ...validGame, finishedAt: null })).not.toThrow();
  });

  it("accepts a waiting (unaccepted invite) game", () => {
    expect(() => gameSchema.parse({ ...validGame, status: "waiting" })).not.toThrow();
  });

  it("accepts ratings, with the delta only once finished", () => {
    const ratings = {
      category: "bullet",
      white: { rating: 1200, win: 18, draw: 8, loss: -2 },
      black: { rating: 1600, win: 2, draw: -8, loss: -18, delta: -18 },
    };
    expect(gameSchema.parse({ ...validGame, ratings }).ratings).toEqual(ratings);
  });

  it("rejects an unknown rating category", () => {
    const side = { rating: 1200, win: 20, draw: 0, loss: -20 };
    expect(() => gameSchema.parse({ ...validGame, ratings: { category: "classical", white: side, black: side } })).toThrow();
  });

  it("rejects a status outside the known set", () => {
    expect(() => gameSchema.parse({ ...validGame, status: "paused" })).toThrow();
  });

  it("rejects clocks sent as strings", () => {
    expect(() =>
      gameSchema.parse({ ...validGame, whiteTimeRemainingMs: "180000" }),
    ).toThrow();
  });

  it.each(["id", "fen", "status", "whiteTimeRemainingMs", "lastMoveAt"])(
    "rejects a payload missing %s",
    (field) => {
      const incomplete: Record<string, unknown> = { ...validGame };
      delete incomplete[field];

      expect(() => gameSchema.parse(incomplete)).toThrow();
    },
  );
});

describe("moveSchema", () => {
  it("accepts a move without a promotion", () => {
    const parsed = moveSchema.parse(validMove);

    expect(parsed.san).toBe("e4");
    expect(parsed.promotion).toBeUndefined();
  });

  it("accepts a promotion", () => {
    expect(moveSchema.parse({ ...validMove, promotion: "queen" }).promotion).toBe("queen");
  });

  it("rejects a colour outside white/black", () => {
    expect(() => moveSchema.parse({ ...validMove, color: "red" })).toThrow();
  });

  it("rejects a ply sent as a string", () => {
    expect(() => moveSchema.parse({ ...validMove, ply: "1" })).toThrow();
  });
});

describe("movesResponseSchema", () => {
  // The endpoint returns a bare array, not an object wrapper.
  it("accepts an array of moves", () => {
    expect(movesResponseSchema.parse([validMove, { ...validMove, ply: 2, color: "black" }])).toHaveLength(2);
  });

  it("accepts the empty array a fresh game returns", () => {
    expect(movesResponseSchema.parse([])).toEqual([]);
  });

  it("rejects an object wrapper", () => {
    expect(() => movesResponseSchema.parse({ moves: [validMove] })).toThrow();
  });
});

describe("serverMessageSchema", () => {
  it("parses game.move_made", () => {
    const parsed = serverMessageSchema.parse({
      type: "game.move_made",
      payload: { game: validGame, move: validMove },
    });

    expect(parsed.type).toBe("game.move_made");
    if (parsed.type === "game.move_made") {
      expect(parsed.payload.move.san).toBe("e4");
    }
  });

  it("parses game.game_over", () => {
    const parsed = serverMessageSchema.parse({
      type: "game.game_over",
      payload: { game: { ...validGame, status: "finished", result: "draw" } },
    });

    expect(parsed.type).toBe("game.game_over");
  });

  it("parses matchmaking.matched", () => {
    const parsed = serverMessageSchema.parse({
      type: "matchmaking.matched",
      payload: { gameId: "game-1", opponent: { username: "bob", rating: 1337 } },
    });

    if (parsed.type === "matchmaking.matched") {
      expect(parsed.payload.opponent.username).toBe("bob");
    }
  });

  it("parses invite.sent and invite.received", () => {
    const sent = serverMessageSchema.parse({
      type: "invite.sent",
      payload: {
        gameId: "game-7",
        to: { userId: "u-bob", username: "bob", rating: 1300 },
        preset: "3+2",
        expiresAt: 1_700_000_060_000,
      },
    });
    const received = serverMessageSchema.parse({
      type: "invite.received",
      payload: {
        gameId: "game-7",
        from: { userId: "u-alice", username: "alice", rating: 1500 },
        preset: "3+2",
        expiresAt: 1_700_000_060_000,
      },
    });

    expect(sent.type).toBe("invite.sent");
    if (received.type === "invite.received") {
      expect(received.payload.from.username).toBe("alice");
    }
  });

  it("parses invite.accepted with the matchmaking.matched shape", () => {
    const parsed = serverMessageSchema.parse({
      type: "invite.accepted",
      payload: { gameId: "game-7", opponent: { username: "bob", rating: 1300 } },
    });

    expect(parsed.type).toBe("invite.accepted");
  });

  it.each(["declined", "cancelled", "expired", "unavailable"])(
    "parses invite.closed with reason %s",
    (reason) => {
      expect(() =>
        serverMessageSchema.parse({ type: "invite.closed", payload: { gameId: "game-7", reason } }),
      ).not.toThrow();
    },
  );

  it("rejects an invite.closed reason outside the known set", () => {
    expect(() =>
      serverMessageSchema.parse({ type: "invite.closed", payload: { gameId: "game-7", reason: "bored" } }),
    ).toThrow();
  });

  const dm = { id: 42, conversationId: "conv-1", senderId: "u-alice", body: "hi", sentAt: 1_700_000_000_000 };
  const conversation = {
    id: "conv-1",
    other: { userId: "u-alice", username: "alice", rating: 1500 },
    folder: "unread",
    unread: true,
    lastMessage: dm,
  };

  it("parses chat.direct_message", () => {
    const parsed = serverMessageSchema.parse({ type: "chat.direct_message", payload: { conversation, message: dm } });

    if (parsed.type === "chat.direct_message") {
      expect(parsed.payload.conversation.folder).toBe("unread");
      expect(parsed.payload.message.body).toBe("hi");
    }
  });

  it("accepts a conversation with no messages yet", () => {
    expect(() =>
      serverMessageSchema.parse({
        type: "chat.direct_message",
        payload: { conversation: { ...conversation, lastMessage: null }, message: dm },
      }),
    ).not.toThrow();
  });

  it("rejects a folder outside inbox/unread/archived", () => {
    expect(() =>
      serverMessageSchema.parse({
        type: "chat.direct_message",
        payload: { conversation: { ...conversation, folder: "spam" }, message: dm },
      }),
    ).toThrow();
  });

  it("parses chat.game_message", () => {
    const parsed = serverMessageSchema.parse({
      type: "chat.game_message",
      payload: {
        gameId: "game-1",
        message: { id: 7, gameId: "game-1", senderId: "u-bob", body: "gg", sentAt: 1_700_000_000_000 },
      },
    });

    expect(parsed.type).toBe("chat.game_message");
  });

  it("keeps an error's inReplyTo", () => {
    const parsed = serverMessageSchema.parse({
      type: "error",
      payload: { code: "rate_limited", message: "slow down", inReplyTo: "chat.send_direct" },
    });

    if (parsed.type === "error") {
      expect(parsed.payload.inReplyTo).toBe("chat.send_direct");
    }
  });

  it("parses the error envelope", () => {
    const parsed = serverMessageSchema.parse({
      type: "error",
      payload: { code: "illegal_move", message: "illegal move" },
    });

    if (parsed.type === "error") {
      expect(parsed.payload.code).toBe("illegal_move");
    }
  });

  it("parses the connection-state messages", () => {
    for (const type of ["game.opponent_disconnected", "game.opponent_reconnected"]) {
      expect(() =>
        serverMessageSchema.parse({ type, payload: { gameId: "game-1" } }),
      ).not.toThrow();
    }
  });

  it("rejects an unknown message type", () => {
    expect(() =>
      serverMessageSchema.parse({ type: "game.exploded", payload: {} }),
    ).toThrow();
  });

  it("rejects a known type carrying the wrong payload", () => {
    expect(() =>
      serverMessageSchema.parse({ type: "game.move_made", payload: { game: validGame } }),
    ).toThrow();
  });

  it("rejects a message with no type", () => {
    expect(() => serverMessageSchema.parse({ payload: {} })).toThrow();
  });
});
