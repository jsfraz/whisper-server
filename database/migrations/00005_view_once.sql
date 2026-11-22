-- +goose Up
-- When the receiver opened the attachment. The server cannot tell a view-once
-- message from a text one (that flag lives inside the ciphertext), so this is a
-- client-reported instant: the receiver calls it once, both sides learn it. The
-- row, and with it the tombstone, lives until expires_at as usual.
ALTER TABLE messages ADD COLUMN opened_at timestamptz;

-- The mutation log gains the matching kind, so both sides hear the open from a
-- live frame and from sync.
ALTER TABLE message_events DROP CONSTRAINT message_events_kind_allowed;
ALTER TABLE message_events ADD CONSTRAINT message_events_kind_allowed
    CHECK (kind IN ('edit', 'delete', 'reaction', 'pin', 'open'));

-- +goose Down
ALTER TABLE message_events DROP CONSTRAINT message_events_kind_allowed;
ALTER TABLE message_events ADD CONSTRAINT message_events_kind_allowed
    CHECK (kind IN ('edit', 'delete', 'reaction', 'pin'));
ALTER TABLE messages DROP COLUMN IF EXISTS opened_at;
