-- Add sold/available state and nullable user_id for unique buyer constraint.
ALTER TABLE tickets
    ADD COLUMN state VARCHAR(16) NOT NULL DEFAULT 'available' AFTER sold_at;

UPDATE tickets SET state = 'sold' WHERE user_id IS NOT NULL AND user_id != 0;

UPDATE tickets SET user_id = NULL WHERE user_id = 0 OR state = 'available';

ALTER TABLE tickets MODIFY user_id BIGINT NULL;

-- 001 created a non-unique index with this name; replace it for one-user-one-booking.
ALTER TABLE tickets DROP INDEX idx_tickets_user_id;
CREATE UNIQUE INDEX idx_tickets_user_id ON tickets (user_id);
