import { useEffect, useMemo, useRef, useState } from "react";
import {
  ActionIcon,
  Alert,
  Badge,
  Button,
  Card,
  Code,
  Collapse,
  Grid,
  Group,
  Stack,
  Tabs,
  Text,
  Title,
  Tooltip,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import {
  IconArrowLeft,
  IconChevronDown,
  IconChevronUp,
  IconCode,
} from "@tabler/icons-react";
import { useNavigate, useParams } from "react-router-dom";

import { templatesApi } from "../api/templates";
import type { CreateTemplateVersionRequest, TemplateVersion } from "../api/types";
import { JsonEditor } from "../components/JsonEditor";
import { HtmlPreview } from "../components/HtmlPreview";
import {
  DroppableTextarea,
  type DroppableTextareaHandle,
} from "../components/DroppableTextarea";
import {
  DroppableTextInput,
  type DroppableTextInputHandle,
} from "../components/DroppableTextInput";
import { ParamChips } from "../components/ParamChips";
import { ParamsBuilder } from "../components/ParamsBuilder";
import { newParam, type Param } from "../params/types";
import { paramsToSchema, schemaToParams } from "../params/schema";
import { collectRefs } from "../params/refs";

const DEFAULT_BODY_HTML = `<!DOCTYPE html>
<html>
  <body style="font-family: sans-serif;">
    <p>Hi <b>{{.name}}</b>,</p>
    <p>Welcome to Copium!</p>
  </body>
</html>`;

const DEFAULT_BODY_TEXT = `Hi {{.name}},\n\nWelcome to Copium!`;

// Pure client-side `text/template`-style placeholder substitution. Lossy
// but good enough for a live preview - the server is the source of truth
// and re-renders the email for real before queueing it. We replace
// `{{.field}}` and `{{ .field }}` with values from the sample params;
// missing keys are left untouched so the user sees them in the preview.
function renderPreview(template: string, params: Record<string, unknown>): string {
  return template.replace(/\{\{\s*\.(\w+)\s*\}\}/g, (_match, key) => {
    const v = params[key];
    if (v === undefined) return `{{.${key}}}`;
    if (typeof v === "boolean") return v ? "yes" : "no";
    return String(v);
  });
}

interface Props {
  mode: "new" | "view";
}

type ActiveField = "subject" | "html" | "text";

export function VersionEditorPage({ mode }: Props) {
  const { id = "", version: versionParam } = useParams();
  const navigate = useNavigate();

  const [subject, setSubject] = useState(mode === "new" ? "Welcome, {{.name}}!" : "");
  const [bodyHtml, setBodyHtml] = useState(mode === "new" ? DEFAULT_BODY_HTML : "");
  const [bodyText, setBodyText] = useState(mode === "new" ? DEFAULT_BODY_TEXT : "");
  const [fromAddress, setFromAddress] = useState("");
  const [params, setParams] = useState<Param[]>(
    mode === "new"
      ? [
          newParam({
            name: "name",
            kind: "text",
            required: true,
            description: "Recipient's first name",
            sample: "Alex",
          }),
        ]
      : [],
  );
  const [submitting, setSubmitting] = useState(false);
  const [loaded, setLoaded] = useState<TemplateVersion | null>(null);
  const [activeField, setActiveField] = useState<ActiveField>("subject");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [advancedJson, setAdvancedJson] = useState<string>("");
  const [advancedEnabled, setAdvancedEnabled] = useState(false);
  const readOnly = mode === "view";

  const subjectRef = useRef<DroppableTextInputHandle | null>(null);
  const htmlRef = useRef<DroppableTextareaHandle | null>(null);
  const textRef = useRef<DroppableTextareaHandle | null>(null);

  useEffect(() => {
    if (mode !== "view" || !versionParam) return;
    void (async () => {
      const v = await templatesApi.getVersion(id, Number(versionParam));
      setLoaded(v);
      setSubject(v.subject);
      setBodyHtml(v.body_html);
      setBodyText(v.body_text ?? "");
      setFromAddress(v.from_address ?? "");
      setParams(schemaToParams(v.params_schema));
    })();
  }, [id, mode, versionParam]);

  const refs = useMemo(() => collectRefs(subject, bodyHtml, bodyText), [subject, bodyHtml, bodyText]);

  const sample = useMemo(() => paramsToSchema(params).sample, [params]);
  const generatedSchema = useMemo(() => paramsToSchema(params).schema, [params]);

  // Keep the advanced JSON view in sync when it's collapsed / not being
  // edited so the user sees the live spec without surprises.
  useEffect(() => {
    if (!advancedEnabled) {
      setAdvancedJson(JSON.stringify(generatedSchema, null, 2));
    }
  }, [generatedSchema, advancedEnabled]);

  const undefinedRefs = useMemo(() => {
    const known = new Set(params.map((p) => p.name));
    return [...refs].filter((r) => !known.has(r));
  }, [refs, params]);

  const previewSubject = useMemo(() => renderPreview(subject, sample), [subject, sample]);
  const previewHtml = useMemo(() => renderPreview(bodyHtml, sample), [bodyHtml, sample]);

  function insertIntoActive(name: string) {
    const snippet = `{{.${name}}}`;
    if (activeField === "subject") subjectRef.current?.insertAtCaret(snippet);
    else if (activeField === "html") htmlRef.current?.insertAtCaret(snippet);
    else textRef.current?.insertAtCaret(snippet);
  }

  async function save() {
    let schemaToSend: Record<string, unknown> = generatedSchema;
    if (advancedEnabled) {
      try {
        schemaToSend = JSON.parse(advancedJson || "{}");
      } catch (err) {
        notifications.show({ color: "red", title: "Invalid schema JSON", message: String(err) });
        return;
      }
    }
    const req: CreateTemplateVersionRequest = {
      subject,
      body_html: bodyHtml,
      body_text: bodyText,
      params_schema: schemaToSend as CreateTemplateVersionRequest["params_schema"],
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

      <Alert variant="light" color="blue">
        Build your template with <b>plain text</b> and add params on the right. Click a chip
        (or drag it) to drop <Code>{"{{.field}}"}</Code> into the subject or body. The
        server validates each send against the schema we generate from your params - no
        JSON required.
      </Alert>

      {undefinedRefs.length > 0 && !readOnly && (
        <Alert color="yellow" title="Undefined params used">
          The template references{" "}
          {undefinedRefs.map((r, i) => (
            <span key={r}>
              <Code>{`{{.${r}}}`}</Code>
              {i < undefinedRefs.length - 1 ? ", " : ""}
            </span>
          ))}{" "}
          but they're not in the params list. Either add them or remove the reference.
        </Alert>
      )}

      <Grid gutter="md">
        {/* Left: editors */}
        <Grid.Col span={{ base: 12, lg: 7 }}>
          <Stack gap="md">
            <Card withBorder padding="sm">
              <Stack gap={6}>
                <ParamChips
                  params={params}
                  usedNames={refs}
                  onInsert={(n) => {
                    setActiveField("subject");
                    subjectRef.current?.focus();
                    subjectRef.current?.insertAtCaret(`{{.${n}}}`);
                  }}
                  hint="Click a chip to insert it into Subject (or drag it anywhere)"
                />
                <DroppableTextInput
                  ref={subjectRef}
                  label="Subject"
                  value={subject}
                  readOnly={readOnly}
                  onChange={setSubject}
                  onFocusChange={(f) => f && setActiveField("subject")}
                />
              </Stack>
            </Card>

            <Card withBorder padding="sm">
              <Stack gap={6}>
                <Group justify="space-between">
                  <Text fw={500} size="sm">
                    Body
                  </Text>
                  <ParamChips
                    params={params}
                    usedNames={refs}
                    onInsert={(n) => insertIntoActive(n)}
                  />
                </Group>
                <Tabs defaultValue="html" onChange={(v) => v && setActiveField(v as ActiveField)}>
                  <Tabs.List>
                    <Tabs.Tab value="html">HTML</Tabs.Tab>
                    <Tabs.Tab value="text">Plain text</Tabs.Tab>
                  </Tabs.List>

                  <Tabs.Panel value="html" pt="sm">
                    <DroppableTextarea
                      ref={htmlRef}
                      value={bodyHtml}
                      readOnly={readOnly}
                      onChange={setBodyHtml}
                      onFocusChange={(f) => f && setActiveField("html")}
                      autosize
                      minRows={14}
                      styles={{ input: { fontFamily: "var(--mantine-font-family-monospace)" } }}
                    />
                  </Tabs.Panel>

                  <Tabs.Panel value="text" pt="sm">
                    <DroppableTextarea
                      ref={textRef}
                      value={bodyText}
                      readOnly={readOnly}
                      onChange={setBodyText}
                      onFocusChange={(f) => f && setActiveField("text")}
                      autosize
                      minRows={10}
                      styles={{ input: { fontFamily: "var(--mantine-font-family-monospace)" } }}
                    />
                  </Tabs.Panel>
                </Tabs>
              </Stack>
            </Card>

            <Card withBorder padding="sm">
              <DroppableTextInput
                value={fromAddress}
                readOnly={readOnly}
                onChange={setFromAddress}
                label="From address"
                description="Optional - falls back to EMAIL_DEFAULT_FROM"
                placeholder="hello@example.com"
              />
            </Card>
          </Stack>
        </Grid.Col>

        {/* Right: live preview + params builder */}
        <Grid.Col span={{ base: 12, lg: 5 }}>
          <Stack gap="md">
            <Card withBorder padding="sm">
              <Stack gap="xs">
                <Group justify="space-between">
                  <Text fw={500} size="sm">
                    Live preview
                  </Text>
                  <Badge variant="light" size="xs">
                    rendered with sample values
                  </Badge>
                </Group>
                <div>
                  <Text size="xs" c="dimmed">
                    Subject
                  </Text>
                  <Text>{previewSubject || <i>(empty)</i>}</Text>
                </div>
                <HtmlPreview html={previewHtml} height={360} />
              </Stack>
            </Card>

            <Card withBorder padding="sm">
              <ParamsBuilder
                params={params}
                onChange={readOnly ? () => {} : setParams}
                usedNames={refs}
              />
            </Card>

            <Card withBorder padding="sm">
              <Stack gap="xs">
                <Group justify="space-between">
                  <Group gap="xs">
                    <IconCode size={16} />
                    <Text fw={500} size="sm">
                      Advanced: raw JSON Schema
                    </Text>
                  </Group>
                  <Tooltip label={showAdvanced ? "Hide" : "Show"}>
                    <ActionIcon variant="subtle" onClick={() => setShowAdvanced((s) => !s)}>
                      {showAdvanced ? <IconChevronUp size={16} /> : <IconChevronDown size={16} />}
                    </ActionIcon>
                  </Tooltip>
                </Group>
                <Collapse in={showAdvanced}>
                  <Stack gap="xs">
                    <Text size="xs" c="dimmed">
                      Generated from the params above. Toggle "Edit raw JSON" if you need a
                      schema feature the visual builder doesn't expose (nested objects,
                      <Code>oneOf</Code>, custom keywords...). The visual editor stops syncing
                      while raw mode is on.
                    </Text>
                    <Group>
                      <Button
                        size="xs"
                        variant={advancedEnabled ? "filled" : "default"}
                        onClick={() => setAdvancedEnabled((v) => !v)}
                        disabled={readOnly}
                      >
                        {advancedEnabled ? "Disable raw mode" : "Edit raw JSON"}
                      </Button>
                    </Group>
                    <JsonEditor
                      value={advancedJson}
                      onChange={advancedEnabled ? setAdvancedJson : () => {}}
                      minRows={8}
                    />
                  </Stack>
                </Collapse>
              </Stack>
            </Card>
          </Stack>
        </Grid.Col>
      </Grid>
    </Stack>
  );
}
