import { useEffect, useMemo, useState } from "react";
import {
  Button,
  Code,
  Grid,
  Group,
  Stack,
  Tabs,
  Text,
  Textarea,
  TextInput,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconArrowLeft } from "@tabler/icons-react";
import { useNavigate, useParams } from "react-router-dom";

import { templatesApi } from "../api/templates";
import type { CreateTemplateVersionRequest, TemplateVersion } from "../api/types";
import { JsonEditor } from "../components/JsonEditor";
import { HtmlPreview } from "../components/HtmlPreview";

const DEFAULT_SCHEMA = JSON.stringify(
  {
    type: "object",
    required: ["name"],
    properties: {
      name: { type: "string", description: "Recipient first name" },
    },
  },
  null,
  2,
);

const DEFAULT_BODY_HTML = `<!DOCTYPE html>
<html>
  <body>
    <p>Hi {{.name}},</p>
    <p>Welcome to Copium!</p>
  </body>
</html>`;

const DEFAULT_BODY_TEXT = `Hi {{.name}},\n\nWelcome to Copium!`;

const DEFAULT_SAMPLE = JSON.stringify({ name: "Alex" }, null, 2);

// Pure client-side `text/template`-style placeholder substitution. This is
// intentionally a tiny, lossy approximation - the server is the source of
// truth and renders the email for real before queueing it. We replace
// `{{.field}}` and `{{ .field }}` with values from the sample params; missing
// keys are left untouched so the user can spot them in the preview.
function renderPreview(template: string, params: Record<string, unknown>): string {
  return template.replace(/\{\{\s*\.(\w+)\s*\}\}/g, (_match, key) => {
    const v = params[key];
    if (v === undefined) return `{{.${key}}}`;
    return String(v);
  });
}

interface Props {
  mode: "new" | "view";
}

export function VersionEditorPage({ mode }: Props) {
  const { id = "", version: versionParam } = useParams();
  const navigate = useNavigate();

  const [subject, setSubject] = useState(mode === "new" ? "Welcome, {{.name}}!" : "");
  const [bodyHtml, setBodyHtml] = useState(mode === "new" ? DEFAULT_BODY_HTML : "");
  const [bodyText, setBodyText] = useState(mode === "new" ? DEFAULT_BODY_TEXT : "");
  const [fromAddress, setFromAddress] = useState("");
  const [schemaText, setSchemaText] = useState(mode === "new" ? DEFAULT_SCHEMA : "{}");
  const [sampleText, setSampleText] = useState(DEFAULT_SAMPLE);
  const [submitting, setSubmitting] = useState(false);
  const [loaded, setLoaded] = useState<TemplateVersion | null>(null);
  const readOnly = mode === "view";

  useEffect(() => {
    if (mode !== "view" || !versionParam) return;
    void (async () => {
      const v = await templatesApi.getVersion(id, Number(versionParam));
      setLoaded(v);
      setSubject(v.subject);
      setBodyHtml(v.body_html);
      setBodyText(v.body_text ?? "");
      setFromAddress(v.from_address ?? "");
      setSchemaText(JSON.stringify(v.params_schema ?? {}, null, 2));
    })();
  }, [id, mode, versionParam]);

  const sampleParams = useMemo<Record<string, unknown>>(() => {
    try {
      return JSON.parse(sampleText || "{}");
    } catch {
      return {};
    }
  }, [sampleText]);

  const previewSubject = useMemo(
    () => renderPreview(subject, sampleParams),
    [subject, sampleParams],
  );
  const previewHtml = useMemo(
    () => renderPreview(bodyHtml, sampleParams),
    [bodyHtml, sampleParams],
  );

  async function save() {
    let parsedSchema: Record<string, unknown>;
    try {
      parsedSchema = JSON.parse(schemaText || "{}");
    } catch (err) {
      notifications.show({ color: "red", title: "Invalid schema JSON", message: String(err) });
      return;
    }
    const req: CreateTemplateVersionRequest = {
      subject,
      body_html: bodyHtml,
      body_text: bodyText,
      params_schema: parsedSchema as CreateTemplateVersionRequest["params_schema"],
      from_address: fromAddress,
    };
    setSubmitting(true);
    try {
      const v = await templatesApi.createVersion(id, req);
      notifications.show({ color: "green", title: "Version created", message: `v${v.version}` });
      navigate(`/templates/${id}`);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Stack>
      <Group justify="space-between">
        <Group>
          <Button
            variant="default"
            leftSection={<IconArrowLeft size={16} />}
            onClick={() => navigate(`/templates/${id}`)}
          >
            Back
          </Button>
          <Title order={3}>
            {mode === "new" ? "New version" : `Version v${loaded?.version ?? versionParam}`}
          </Title>
        </Group>
        {mode === "new" && (
          <Button onClick={save} loading={submitting}>
            Save version
          </Button>
        )}
      </Group>

      <Text c="dimmed" size="sm">
        Templates use Go's <Code>text/template</Code> syntax. Reference fields with{" "}
        <Code>{"{{.field}}"}</Code>. The preview here is a quick approximation - the server
        re-renders the message before it gets sent.
      </Text>

      <Grid gutter="md">
        <Grid.Col span={{ base: 12, lg: 6 }}>
          <Stack>
            <TextInput
              label="Subject"
              value={subject}
              readOnly={readOnly}
              onChange={(e) => setSubject(e.currentTarget.value)}
            />
            <TextInput
              label="From address"
              description="Optional - falls back to EMAIL_DEFAULT_FROM"
              value={fromAddress}
              readOnly={readOnly}
              onChange={(e) => setFromAddress(e.currentTarget.value)}
              placeholder="hello@example.com"
            />

            <Tabs defaultValue="html">
              <Tabs.List>
                <Tabs.Tab value="html">Body (HTML)</Tabs.Tab>
                <Tabs.Tab value="text">Body (text)</Tabs.Tab>
                <Tabs.Tab value="schema">Params schema</Tabs.Tab>
                <Tabs.Tab value="sample">Sample params</Tabs.Tab>
              </Tabs.List>

              <Tabs.Panel value="html" pt="sm">
                <Textarea
                  value={bodyHtml}
                  readOnly={readOnly}
                  onChange={(e) => setBodyHtml(e.currentTarget.value)}
                  autosize
                  minRows={14}
                  styles={{ input: { fontFamily: "var(--mantine-font-family-monospace)" } }}
                />
              </Tabs.Panel>

              <Tabs.Panel value="text" pt="sm">
                <Textarea
                  value={bodyText}
                  readOnly={readOnly}
                  onChange={(e) => setBodyText(e.currentTarget.value)}
                  autosize
                  minRows={10}
                  styles={{ input: { fontFamily: "var(--mantine-font-family-monospace)" } }}
                />
              </Tabs.Panel>

              <Tabs.Panel value="schema" pt="sm">
                <JsonEditor
                  description="JSON Schema (draft 2020-12) used to validate params before send."
                  value={schemaText}
                  onChange={readOnly ? () => {} : setSchemaText}
                  minRows={12}
                />
              </Tabs.Panel>

              <Tabs.Panel value="sample" pt="sm">
                <JsonEditor
                  description="Sample params used only for the live preview (not persisted)."
                  value={sampleText}
                  onChange={setSampleText}
                  minRows={10}
                />
              </Tabs.Panel>
            </Tabs>
          </Stack>
        </Grid.Col>

        <Grid.Col span={{ base: 12, lg: 6 }}>
          <Stack>
            <div>
              <Text size="sm" fw={500}>
                Preview subject
              </Text>
              <Text>{previewSubject || <i>(empty)</i>}</Text>
            </div>
            <div>
              <Text size="sm" fw={500} mb="xs">
                Preview HTML
              </Text>
              <HtmlPreview html={previewHtml} height={520} />
            </div>
          </Stack>
        </Grid.Col>
      </Grid>
    </Stack>
  );
}
