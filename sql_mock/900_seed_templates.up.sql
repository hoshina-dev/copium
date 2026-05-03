-- Sample template + version for local dev.
INSERT INTO email_templates (id, code, name, description)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'welcome_email',
    'Welcome',
    'Greet a newly created user.'
);

INSERT INTO email_template_versions (id, template_id, version, subject, body_html, body_text, params_schema, from_address)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    1,
    'Welcome to Hoshina, {{.name}}!',
    '<p>Hi {{.name}}, glad you joined.</p>',
    'Hi {{.name}}, glad you joined.',
    '{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}'::jsonb,
    'Hoshina <noreply@hoshina.dev>'
);

UPDATE email_templates
SET active_version_id = '00000000-0000-0000-0000-000000000002'
WHERE id = '00000000-0000-0000-0000-000000000001';
