CREATE TABLE conversations (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_lo         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_hi         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at      timestamptz NOT NULL DEFAULT now(),
    last_message_at timestamptz NOT NULL DEFAULT now(),
    CHECK (user_lo < user_hi),
    UNIQUE (user_lo, user_hi) -- one conversation per pair
);

-- Per-user state. A conversation's folder and unread flag are derived
-- from these columns rather than stored.
CREATE TABLE conversation_members (
    conversation_id uuid NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    replied         boolean NOT NULL DEFAULT false, -- has sent at least one message
    archived_at     timestamptz,
    last_read_at    timestamptz,
    PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX conversation_members_user_idx ON conversation_members (user_id);

CREATE TABLE direct_messages (
    id              bigserial PRIMARY KEY, -- doubles as the paging cursor
    conversation_id uuid NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body            text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX direct_messages_conversation_idx ON direct_messages (conversation_id, id DESC);

CREATE TABLE game_messages (
    id         bigserial PRIMARY KEY,
    game_id    uuid NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    sender_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX game_messages_game_idx ON game_messages (game_id, id);
