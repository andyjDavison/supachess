CREATE TABLE friendships (
    requester_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       text NOT NULL CHECK (status IN ('pending', 'accepted')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    accepted_at  timestamptz,
    CHECK (requester_id <> addressee_id),
    PRIMARY KEY (requester_id, addressee_id)
);

-- One row per unordered pair, so A->B and B->A can't both exist.
CREATE UNIQUE INDEX friendships_pair_key
    ON friendships (LEAST(requester_id, addressee_id), GREATEST(requester_id, addressee_id));
CREATE INDEX friendships_addressee_idx ON friendships (addressee_id);

CREATE TABLE blocks (
    blocker_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (blocker_id <> blocked_id),
    PRIMARY KEY (blocker_id, blocked_id)
);
CREATE INDEX blocks_blocked_idx ON blocks (blocked_id);

-- Prefix search on username.
CREATE INDEX users_username_lower_prefix_idx ON users (lower(username) text_pattern_ops);
