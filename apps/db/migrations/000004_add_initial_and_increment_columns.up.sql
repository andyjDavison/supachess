DROP TABLE IF EXISTS games;

CREATE TABLE games (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    white_id      uuid NOT NULL,
    black_id      uuid NOT NULL,
    fen text NOT NULL,
    game_status text NOT NULL,
    result text,
    result_reason text,
    white_time      INTERVAL,
    black_time      INTERVAL,
    initial    INTERVAL,
    increment INTERVAL,
    last_move_at    timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL,
    finished_at timestamptz
);