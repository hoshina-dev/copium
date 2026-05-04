-- Direct-email dispatch: callers can now POST /emails/send with a
-- to_address instead of a custapi user_id. The outbox snapshot still
-- carries the resolved address; it just no longer needs a user reference
-- when the email was sent to an external recipient.
ALTER TABLE email_outbox
    ALTER COLUMN user_id DROP NOT NULL;
