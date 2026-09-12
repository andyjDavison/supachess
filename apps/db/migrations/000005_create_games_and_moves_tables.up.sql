DROP TABLE games;

CREATE TABLE games (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    white_id       uuid NOT NULL REFERENCES users(id),
    black_id       uuid NOT NULL REFERENCES users(id),
    fen            text NOT NULL,
    game_status    text NOT NULL,
    result         text NOT NULL DEFAULT '',
    result_reason  text NOT NULL DEFAULT '',
    white_time     bigint NOT NULL,
    black_time     bigint NOT NULL,
    initial        bigint NOT NULL,
    increment      bigint NOT NULL,
    last_move_at   timestamptz NOT NULL DEFAULT now(),
    created_at     timestamptz NOT NULL DEFAULT now(),
    finished_at    timestamptz
);

CREATE TABLE moves (
    game_id     uuid NOT NULL REFERENCES games(id),
    ply         integer NOT NULL,
    color       text NOT NULL,
    from_square text NOT NULL,
    to_square   text NOT NULL,
    promotion   text NOT NULL DEFAULT '',
    san         text NOT NULL,
    fen_after   text NOT NULL,
    played_at   timestamptz NOT NULL,
    PRIMARY KEY (game_id, ply)
);