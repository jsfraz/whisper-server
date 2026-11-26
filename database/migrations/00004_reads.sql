-- +goose Up
-- Read receipts. One cursor per account and conversation: the server learns
-- the instant up to which a chat was read, never which message was opened.
-- Opting in is a client-side per-chat switch, so an account that keeps it off
-- simply never writes a row.
CREATE TABLE conversation_reads (
    user_id text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    conversation text NOT NULL,
    -- The other participant. Reading the cursors someone else set in my chats
    -- is one indexed lookup by this column.
    peer_id text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    read_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, conversation)
);

CREATE INDEX idx_conversation_reads_peer ON conversation_reads (peer_id);

-- +goose Down
DROP TABLE IF EXISTS conversation_reads;
