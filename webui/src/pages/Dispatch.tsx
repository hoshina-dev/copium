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
import type { JSONObject, OutboxRow, Template, TemplateVersion } from "../api/types";
import { SchemaForm } from "../components/SchemaForm";

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
  const [userId, setUserId] = useState("");
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

  async function send() {
    if (!selected) return;
    setSubmitting(true);
    setOutbox(null);
    try {
      const res = await emailsApi.send({
        template_id: selected,
        user_id: userId,
        params,
      });
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
        Pick a template, supply a Custapi user id and the template parameters, then send.
        The recipient address is resolved server-side from <Code>custapi</Code>.
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

            <TextInput
              label="User ID"
              description="Custapi user UUID. The server resolves the email address from this id."
              placeholder="00000000-0000-0000-0000-000000000000"
              value={userId}
              onChange={(e) => setUserId(e.currentTarget.value)}
            />

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
                disabled={!selected || !userId || !activeVersion}
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
