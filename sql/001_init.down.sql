DROP TABLE IF EXISTS email_outbox;
ALTER TABLE IF EXISTS email_templates DROP CONSTRAINT IF EXISTS fk_active_version;
DROP TABLE IF EXISTS email_template_versions;
DROP TABLE IF EXISTS email_templates;
