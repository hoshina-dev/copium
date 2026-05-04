import { useEffect, useMemo, useRef, useState } from "react";
import {
  Alert,
  Badge,
  Button,
  Card,
  Code,
  Divider,
  Grid,
  Group,
  SegmentedControl,
  Select,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconSend } from "@tabler/icons-react";
import { useNavigate, useParams } from "react-router-dom";

import { templatesApi } from "../api/templates";
import { emailsApi } from "../api/emails";
import type {
  JSONObject,
  OutboxRow,
  SendEmailRequest,
  Template,
  TemplateVersion,
} from "../api/types";
import { SchemaForm } from "../components/SchemaForm";

type RecipientMode = "user" | "direct";

// Quick-and-dirty client-side sanity check. The server uses net/mail to do
// the authoritative validation; we just want to disable Send for obvious
// typos so users get instant feedback.
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

const STATUS_COLOR: Record<string, string> = {
  pending: "gray",
  sending: "blue",
  sent: "green",
  failed: "yellow",
  dead: "red",
};

export function DispatchPage() {
  const { templateId: paramTemplateId } = useParams();
  const navigate = useNavigate();

  const [templates, setTemplates] = useState<Template[]>([]);
  const [selected, setSelected] = useState<string | null>(paramTemplateId ?? null);
  const [activeVersion, setActiveVersion] = useState<TemplateVersion | null>(null);
  const [recipientMode, setRecipientMode] = useState<RecipientMode>("user");
  const [userId, setUserId] = useState("");
  const [toAddress, setToAddress] = useState("");
  const [params, setParams] = useState<JSONObject>({});
  const [submitting, setSubmitting] = useState(false);
  const [outbox, setOutbox] = useState<OutboxRow | null>(null);
  const pollRef = useRef<number | null>(null);

  useEffect(() => {
    void templatesApi.list().then(setTemplates);
  }, []);

  useEffect(() => {
    setActiveVersion(null);
    setParams({});
    if (!selected) return;
    void (async () => {
      const t = await templatesApi.get(selected);
      if (!t.active_version_id) return;
      const versions = await templatesApi.listVersions(selected);
      const av = versions.find((v) => v.id === t.active_version_id);
      if (av) setActiveVersion(av);
    })();
  }, [selected]);

  useEffect(() => {
    return () => {
      if (pollRef.current !== null) window.clearInterval(pollRef.current);
    };
  }, []);

  const selectedTemplate = useMemo(
    () => templates.find((t) => t.id === selected) ?? null,
    [templates, selected],
  );

  function startPolling(outboxId: string) {
    if (pollRef.current !== null) window.clearInterval(pollRef.current);
    pollRef.current = window.setInterval(async () => {
      try {
        const row = await emailsApi.getOutbox(outboxId);
        setOutbox(row);
        if (row.status === "sent" || row.status === "dead") {
          if (pollRef.current !== null) {
            window.clearInterval(pollRef.current);
            pollRef.current = null;
          }
        }
      } catch {
        // notifications already shown by http layer
      }
    }, 1500);
  }

  const recipientValid =
    recipientMode === "user" ? UUID_RE.test(userId) : EMAIL_RE.test(toAddress);

  async function send() {
    if (!selected || !recipientValid) return;
    setSubmitting(true);
    setOutbox(null);
    try {
      const req: SendEmailRequest = {
        template_id: selected,
        params,
        ...(recipientMode === "user"
          ? { user_id: userId }
          : { to_address: toAddress }),
      };
      const res = await emailsApi.send(req);
      notifications.show({
        color: "green",
        title: "Queued",
        message: `outbox ${res.outbox_id}`,
      });
      const row = await emailsApi.getOutbox(res.outbox_id);
      setOutbox(row);
      startPolling(res.outbox_id);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Stack>
      <Title order={2}>Dispatch a test email</Title>
      <Text c="dimmed" size="sm">
        Pick a template, choose a recipient (a <Code>custapi</Code> user we'll resolve to an
        email, or a direct email address for someone outside the system), fill in the
        template parameters, then send.
      </Text>

      <Grid gutter="md">
        <Grid.Col span={{ base: 12, md: 6 }}>
          <Stack>
            <Select
              label="Template"
              placeholder="Choose a template"
              searchable
              data={templates.map((t) => ({
                value: t.id,
                label: `${t.code} - ${t.name}`,
              }))}
              value={selected}
              onChange={(v) => {
                setSelected(v);
                if (v) navigate(`/dispatch/${v}`, { replace: true });
              }}
            />

            {selectedTemplate && !selectedTemplate.active_version_id && (
              <Alert color="yellow" title="No active version">
                This template has no active version yet - set one from its detail page first.
              </Alert>
            )}

            <Card withBorder padding="sm">
              <Stack gap="xs">
                <Text size="sm" fw={500}>
                  Recipient
                </Text>
                <SegmentedControl
                  fullWidth
                  value={recipientMode}
                  onChange={(v) => setRecipientMode(v as RecipientMode)}
                  data={[
                    { value: "user", label: "Custapi user" },
                    { value: "direct", label: "Direct email" },
                  ]}
                />
                {recipientMode === "user" ? (
                  <TextInput
                    label="User ID"
                    description="Custapi user UUID. The server resolves the email address from this id."
                    placeholder="00000000-0000-0000-0000-000000000000"
                    value={userId}
                    onChange={(e) => setUserId(e.currentTarget.value.trim())}
                    error={userId && !UUID_RE.test(userId) ? "Not a valid UUID" : undefined}
                  />
                ) : (
                  <TextInput
                    label="Email address"
                    description="Send straight to this address. Use this for partners or one-offs that aren't in our system."
                    placeholder="someone@example.com"
                    value={toAddress}
                    onChange={(e) => setToAddress(e.currentTarget.value.trim())}
                    error={
                      toAddress && !EMAIL_RE.test(toAddress) ? "Doesn't look like an email" : undefined
                    }
                  />
                )}
              </Stack>
            </Card>

            {activeVersion && (
              <Card withBorder>
                <Stack gap="sm">
                  <Group justify="space-between">
                    <Text fw={500}>Parameters (v{activeVersion.version})</Text>
                    <Badge color="green" variant="light">
                      active
                    </Badge>
                  </Group>
                  <SchemaForm
                    schema={activeVersion.params_schema}
                    value={params}
                    onChange={setParams}
                  />
                </Stack>
              </Card>
            )}

            <Group justify="flex-end">
              <Button
                leftSection={<IconSend size={16} />}
                onClick={send}
                loading={submitting}
                disabled={!selected || !recipientValid || !activeVersion}
              >
                Send
              </Button>
            </Group>
          </Stack>
        </Grid.Col>

        <Grid.Col span={{ base: 12, md: 6 }}>
          {outbox ? (
            <Card withBorder>
              <Stack gap="xs">
                <Group justify="space-between">
                  <Text fw={500}>Outbox status</Text>
                  <Badge color={STATUS_COLOR[outbox.status] ?? "gray"}>{outbox.status}</Badge>
                </Group>
                <Divider />
                <Row label="Outbox ID" value={outbox.id} />
                <Row label="To" value={outbox.to_address} />
                <Row label="Subject" value={outbox.subject} />
                <Row label="Attempts" value={`${outbox.attempts} / ${outbox.max_attempts}`} />
                {outbox.last_error && (
                  <Alert color="red" title="Last error">
                    {outbox.last_error}
                  </Alert>
                )}
                {outbox.sent_at && (
                  <Row label="Sent at" value={new Date(outbox.sent_at).toLocaleString()} />
                )}
                <Group justify="flex-end">
                  <Button
                    variant="default"
                    size="xs"
                    onClick={() => navigate(`/outbox/${outbox.id}`)}
                  >
                    Open detail
                  </Button>
                </Group>
              </Stack>
            </Card>
          ) : (
            <Card withBorder>
              <Text c="dimmed" size="sm">
                The send result will appear here. We'll poll the outbox automatically until
                the status becomes <Code>sent</Code> or <Code>dead</Code>.
              </Text>
            </Card>
          )}
        </Grid.Col>
      </Grid>
    </Stack>
  );
}

function Row(props: { label: string; value: string }) {
  return (
    <Group justify="space-between" wrap="nowrap">
      <Text size="sm" c="dimmed">
        {props.label}
      </Text>
      <Text size="sm" style={{ wordBreak: "break-all" }}>
        {props.value}
      </Text>
    </Group>
  );
}
