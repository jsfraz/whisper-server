-- +goose Up
-- Per-account visibility. The ciphertext row stays shared between both
-- participants, so "delete for me" must never touch the other side's archive:
-- it is an entry here, not a missing message row.
--
-- conversation_hides is the whole-chat cut: everything sent at or before
-- cleared_at is invisible to that account.
CREATE TABLE conversation_hides (
    user_id text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    conversation text NOT NULL,
    cleared_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, conversation)
);

-- message_hides is one message removed for one account. Only the peer's
-- messages need this: an own message deleted for everyone is tombstoned in
-- messages.deleted_at instead.
CREATE TABLE message_hides (
    user_id text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    message_id text NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, message_id)
);

CREATE INDEX idx_message_hides_message ON message_hides (message_id);

-- +goose Down
DROP TABLE IF EXISTS message_hides;
DROP TABLE IF EXISTS conversation_hides;
