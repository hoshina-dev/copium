-- copium: initial schema
-- Enables pgcrypto for gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Logical template; active_version_id points at email_template_versions.
CREATE TABLE email_templates (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    code               text        NOT NULL,
    name               text        NOT NULL,
    description        text,
    active_version_id  uuid,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    deleted_at         timestamptz
);

CREATE UNIQUE INDEX uq_email_templates_code
    ON email_templates(code)
    WHERE deleted_at IS NULL;

-- Immutable version snapshots. (template_id, version) is unique.
CREATE TABLE email_template_versions (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id    uuid        NOT NULL REFERENCES email_templates(id) ON DELETE CASCADE,
    version        integer     NOT NULL,
    subject        text        NOT NULL,
    body_html      text        NOT NULL,
    body_text      text,
    params_schema  jsonb       NOT NULL,
    from_address   text,
    created_by     text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (template_id, version)
);

ALTER TABLE email_templates
    ADD CONSTRAINT fk_active_version
    FOREIGN KEY (active_version_id) REFERENCES email_template_versions(id);

-- Outbox: queued + history. Each row carries a self-contained snapshot.
CREATE TABLE email_outbox (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    template_version_id  uuid        NOT NULL REFERENCES email_template_versions(id),
    user_id              uuid        NOT NULL,
    to_address           text        NOT NULL,
    from_address         text        NOT NULL,
    subject              text        NOT NULL,
    body_html            text        NOT NULL,
    body_text            text,
    params               jsonb       NOT NULL DEFAULT '{}'::jsonb,
    status               text        NOT NULL CHECK (status IN ('queued','sending','sent','failed','dead')),
    attempts             integer     NOT NULL DEFAULT 0,
    max_attempts         integer     NOT NULL DEFAULT 5,
    scheduled_at         timestamptz NOT NULL DEFAULT now(),
    last_error           text,
    provider             text,
    provider_message_id  text,
    sent_at              timestamptz,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_outbox_dispatch ON email_outbox (status, scheduled_at);
CREATE INDEX idx_outbox_user ON email_outbox (user_id);
