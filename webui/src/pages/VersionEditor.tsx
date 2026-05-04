import { useEffect, useMemo, useRef, useState } from "react";
import {
  ActionIcon,
  Alert,
  Badge,
  Button,
  Card,
  Code,
  Grid,
  Group,
  Stack,
  Tabs,
  Text,
  Textarea,
  TextInput,
  Title,
  Tooltip,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconArrowLeft, IconPlayerPlay, IconRefresh } from "@tabler/icons-react";
import { useNavigate, useParams } from "react-router-dom";

import { templatesApi } from "../api/templates";
import type {
  CreateTemplateVersionRequest,
  PreviewTemplateResponse,
  TemplateVersion,
} from "../api/types";
import { HtmlPreview } from "../components/HtmlPreview";
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

interface Props {
  mode: "new" | "view";
}

type ActiveField = "subject" | "html" | "text";

// VersionEditorPage is the write/view form for a single template version.
// It deliberately keeps the interaction model boring: plain inputs, a
// visual params builder, and a "Render on server" button that calls the
// backend preview endpoint so what the operator sees is exactly what a
// real send will emit. There is no client-side placeholder substitution
// — past attempts at that drifted from Go's text/template semantics and
// misled users.
export function VersionEditorPage({ mode }: Props) {
  const { id = "", version: versionParam, baseVersion } = useParams();
  const navigate = useNavigate();
  const readOnly = mode === "view";

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
  const subjectRef = useRef<HTMLInputElement | null>(null);
  const htmlRef = useRef<HTMLTextAreaElement | null>(null);
  const textRef = useRef<HTMLTextAreaElement | null>(null);

  const [preview, setPreview] = useState<PreviewTemplateResponse | null>(null);
  const [previewErr, setPreviewErr] = useState<string>("");
  const [previewing, setPreviewing] = useState(false);

  useEffect(() => {
    if (mode === "view" && versionParam) {
      void (async () => {
        const v = await templatesApi.getVersion(id, Number(versionParam));
        setLoaded(v);
        setSubject(v.subject);
        setBodyHtml(v.body_html);
        setBodyText(v.body_text ?? "");
        setFromAddress(v.from_address ?? "");
        setParams(schemaToParams(v.params_schema));
      })();
      return;
    }
    if (mode === "new" && baseVersion) {
      void (async () => {
        const v = await templatesApi.getVersion(id, Number(baseVersion));
        setLoaded(v);
        setSubject(v.subject);
        setBodyHtml(v.body_html);
        setBodyText(v.body_text ?? "");
        setFromAddress(v.from_address ?? "");
        setParams(schemaToParams(v.params_schema));
      })();
    }
  }, [id, mode, versionParam, baseVersion]);

  const refs = useMemo(() => collectRefs(subject, bodyHtml, bodyText), [subject, bodyHtml, bodyText]);
  const { schema: generatedSchema, sample } = useMemo(() => paramsToSchema(params), [params]);

  const undefinedRefs = useMemo(() => {
    const known = new Set(params.map((p) => p.name));
    return [...refs].filter((r) => !known.has(r));
  }, [refs, params]);

  function insertIntoActive(name: string) {
    const snippet = `{{.${name}}}`;
    const field = activeField === "subject" ? subjectRef.current : activeField === "html" ? htmlRef.current : textRef.current;
    if (!field) return;
    const start = field.selectionStart ?? field.value.length;
    const end = field.selectionEnd ?? start;
    const next = field.value.slice(0, start) + snippet + field.value.slice(end);
    if (activeField === "subject") setSubject(next);
    else if (activeField === "html") setBodyHtml(next);
    else setBodyText(next);
    requestAnimationFrame(() => {
      field.focus();
      const caret = start + snippet.length;
      field.setSelectionRange(caret, caret);
    });
  }

  async function runPreview() {
    setPreviewing(true);
    setPreviewErr("");
    try {
      const out = await templatesApi.preview({
        subject,
        body_html: bodyHtml,
        body_text: bodyText,
        params_schema: generatedSchema as unknown as CreateTemplateVersionRequest["params_schema"],
        params: sample as unknown as CreateTemplateVersionRequest["params_schema"],
      });
      setPreview(out);
    } catch (err) {
      setPreviewErr(err instanceof Error ? err.message : String(err));
    } finally {
      setPreviewing(false);
    }
  }

  async function save() {
    const req: CreateTemplateVersionRequest = {
      subject,
      body_html: bodyHtml,
      body_text: bodyText,
      params_schema: generatedSchema as CreateTemplateVersionRequest["params_schema"],
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
          {mode === "new" && baseVersion && (
            <Badge variant="light" color="blue">
              based on v{baseVersion}
            </Badge>
          )}
        </Group>
        <Group>
          <Button
            variant="default"
            leftSection={<IconPlayerPlay size={16} />}
            onClick={runPreview}
            loading={previewing}
          >
            Render on server
          </Button>
          {mode === "new" && (
            <Button onClick={save} loading={submitting}>
              Save version
            </Button>
          )}
        </Group>
      </Group>

      {undefinedRefs.length > 0 && !readOnly && (
        <Alert color="yellow" title="Missing variables">
          Your email uses{" "}
          {undefinedRefs.map((r, i) => (
            <span key={r}>
              <Code>{`{{.${r}}}`}</Code>
              {i < undefinedRefs.length - 1 ? ", " : ""}
            </span>
          ))}{" "}
          but you haven't added them on the right. Add each variable, or remove the
          mention from your email — otherwise sends will fail.
        </Alert>
      )}

      <Grid gutter="md">
        <Grid.Col span={{ base: 12, lg: 7 }}>
          <Stack gap="md">
            <Card withBorder padding="sm">
              <Stack gap={6}>
                <VariableChips
                  params={params}
                  usedNames={refs}
                  onInsert={insertIntoActive}
                  activeField={activeField}
                />
                <TextInput
                  ref={subjectRef}
                  label="Subject"
                  value={subject}
                  readOnly={readOnly}
                  onChange={(e) => setSubject(e.currentTarget.value)}
                  onFocus={() => setActiveField("subject")}
                />
              </Stack>
            </Card>

            <Card withBorder padding="sm">
              <Stack gap={6}>
                <Text fw={500} size="sm">Body</Text>
                <Tabs defaultValue="html" onChange={(v) => v && setActiveField(v as ActiveField)}>
                  <Tabs.List>
                    <Tabs.Tab value="html">HTML</Tabs.Tab>
                    <Tabs.Tab value="text">Plain-text fallback</Tabs.Tab>
                  </Tabs.List>
                  <Tabs.Panel value="html" pt="sm">
                    <Textarea
                      ref={htmlRef}
                      value={bodyHtml}
                      readOnly={readOnly}
                      onChange={(e) => setBodyHtml(e.currentTarget.value)}
                      onFocus={() => setActiveField("html")}
                      autosize
                      minRows={14}
                      styles={{ input: { fontFamily: "var(--mantine-font-family-monospace)" } }}
                    />
                  </Tabs.Panel>
                  <Tabs.Panel value="text" pt="sm">
                    <Textarea
                      ref={textRef}
                      value={bodyText}
                      readOnly={readOnly}
                      onChange={(e) => setBodyText(e.currentTarget.value)}
                      onFocus={() => setActiveField("text")}
                      autosize
                      minRows={10}
                      styles={{ input: { fontFamily: "var(--mantine-font-family-monospace)" } }}
                    />
                  </Tabs.Panel>
                </Tabs>
              </Stack>
            </Card>

            <Card withBorder padding="sm">
              <TextInput
                label="From address"
                description="Optional — falls back to EMAIL_DEFAULT_FROM"
                placeholder="hello@example.com"
                value={fromAddress}
                readOnly={readOnly}
                onChange={(e) => setFromAddress(e.currentTarget.value)}
              />
            </Card>

            <Card withBorder padding="sm">
              <ParamsBuilder
                params={params}
                onChange={readOnly ? () => {} : setParams}
                usedNames={refs}
              />
            </Card>
          </Stack>
        </Grid.Col>

        <Grid.Col span={{ base: 12, lg: 5 }}>
          <Stack gap="md" style={{ position: "sticky", top: 16 }}>
            <Card withBorder padding="sm">
              <Stack gap="xs">
                <Group justify="space-between">
                  <Text fw={500} size="sm">Preview</Text>
                  <Tooltip label="Re-render with current sample values">
                    <ActionIcon variant="subtle" onClick={runPreview} loading={previewing}>
                      <IconRefresh size={16} />
                    </ActionIcon>
                  </Tooltip>
                </Group>
                {previewErr && (
                  <Alert color="red" variant="light" title="Render failed">
                    {previewErr}
                  </Alert>
                )}
                {!preview && !previewErr && (
                  <Text size="xs" c="dimmed">
                    Click <b>Render on server</b> to see the real rendered output. The
                    server uses the same renderer as live sends, with the sample values
                    from your variables.
                  </Text>
                )}
                {preview && (
                  <>
                    <div>
                      <Text size="xs" c="dimmed">Subject</Text>
                      <Text>{preview.subject || <i>(empty)</i>}</Text>
                    </div>
                    <Tabs defaultValue="html">
                      <Tabs.List>
                        <Tabs.Tab value="html">HTML</Tabs.Tab>
                        <Tabs.Tab value="text">Text</Tabs.Tab>
                      </Tabs.List>
                      <Tabs.Panel value="html" pt="xs">
                        <HtmlPreview html={preview.body_html} height={420} />
                      </Tabs.Panel>
                      <Tabs.Panel value="text" pt="xs">
                        <Textarea
                          value={preview.body_text ?? ""}
                          readOnly
                          autosize
                          minRows={10}
                          styles={{ input: { fontFamily: "var(--mantine-font-family-monospace)" } }}
                        />
                      </Tabs.Panel>
                    </Tabs>
                  </>
                )}
              </Stack>
            </Card>
          </Stack>
        </Grid.Col>
      </Grid>
    </Stack>
  );
}

function VariableChips(props: {
  params: Param[];
  usedNames: Set<string>;
  onInsert: (name: string) => void;
  activeField: ActiveField;
}) {
  const { params, usedNames, onInsert, activeField } = props;
  const named = params.filter((p) => p.name.trim().length > 0);
  if (named.length === 0) return null;
  const target = activeField === "subject" ? "subject" : activeField === "html" ? "HTML body" : "text body";
  return (
    <Group gap={6}>
      <Text size="xs" c="dimmed">
        Click to insert into {target}:
      </Text>
      {named.map((p) => (
        <Badge
          key={p.id}
          variant={usedNames.has(p.name) ? "filled" : "light"}
          color={usedNames.has(p.name) ? "green" : "gray"}
          style={{ cursor: "pointer" }}
          onClick={() => onInsert(p.name)}
        >
          {p.name}
        </Badge>
      ))}
    </Group>
  );
}
