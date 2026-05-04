-- WARNING: this rollback fails if any existing row has user_id IS NULL
-- (eg. emails dispatched directly to to_address). Backfill or delete those
-- rows first.
ALTER TABLE email_outbox
    ALTER COLUMN user_id SET NOT NULL;
