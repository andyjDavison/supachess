-- One rating per player per speed. users.rating stays as the Blitz rating,
-- kept in sync when a Blitz game finishes, for everything that shows a
-- single rating (friends, search, chat, /api/me).
CREATE TABLE user_ratings (
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category     text NOT NULL CHECK (category IN ('bullet', 'blitz', 'rapid')),
    rating       integer NOT NULL,
    games_played integer NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, category)
);

INSERT INTO user_ratings (user_id, category, rating)
SELECT u.id, c.category, u.rating
FROM users u CROSS JOIN (VALUES ('bullet'), ('blitz'), ('rapid')) AS c(category);

-- Snapshot taken when the game is created; deltas filled in when it
-- finishes. NULL on games from before this migration.
ALTER TABLE games
    ADD COLUMN white_rating       integer,
    ADD COLUMN black_rating       integer,
    ADD COLUMN white_k            smallint,
    ADD COLUMN black_k            smallint,
    ADD COLUMN white_rating_delta integer,
    ADD COLUMN black_rating_delta integer;
