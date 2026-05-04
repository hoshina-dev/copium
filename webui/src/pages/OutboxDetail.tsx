import { useEffect, useState } from "react";
import {
  Alert,
  Badge,
  Button,
  Card,
  Code,
  Divider,
  Grid,
  Group,
  Loader,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { IconRefresh, IconSearch } from "@tabler/icons-react";
import { useNavigate, useParams } from "react-router-dom";

import { emailsApi } from "../api/emails";
import type { OutboxRow } from "../api/types";

const STATUS_COLOR: Record<string, string> = {
  pending: "gray",
  sending: "blue",
  sent: "green",
  failed: "yellow",
  dead: "red",
};

export function OutboxDetailPage() {
  const { id: routeId } = useParams();
  const navigate = useNavigate();
  const [lookupId, setLookupId] = useState(routeId ?? "");
  const [row, setRow] = useState<OutboxRow | null>(null);
  const [loading, setLoading] = useState(false);

  async function load(id: string) {
    if (!id) return;
    setLoading(true);
    try {
      setRow(await emailsApi.getOutbox(id));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (routeId) void load(routeId);
  }, [routeId]);

  return (
    <Stack>
      <Title order={2}>Outbox</Title>
      <Group>
        <TextInput
          flex={1}
          placeholder="Outbox UUID"
          value={lookupId}
          onChange={(e) => setLookupId(e.currentTarget.value)}
        />
        <Button
          leftSection={<IconSearch size={16} />}
          onClick={() => navigate(`/outbox/${lookupId}`)}
          disabled={!lookupId}
        >
          Lookup
        </Button>
        <Button
          leftSection={<IconRefresh size={16} />}
          variant="default"
          onClick={() => routeId && load(routeId)}
          loading={loading}
          disabled={!routeId}
        >
          Refresh
        </Button>
      </Group>

      {!routeId ? (
        <Text c="dimmed">
          Paste an outbox id to inspect its current snapshot, status and last error.
        </Text>
      ) : loading && !row ? (
        <Loader />
      ) : row ? (
        <Grid gutter="md">
          <Grid.Col span={{ base: 12, md: 6 }}>
            <Card withBorder>
              <Stack gap="xs">
                <Group justify="space-between">
                  <Text fw={500}>Status</Text>
                  <Badge color={STATUS_COLOR[row.status] ?? "gray"}>{row.status}</Badge>
                </Group>
                <Divider />
                <Row label="ID" value={row.id} />
                <Row label="Template version" value={row.template_version_id} />
                <Row label="User" value={row.user_id ?? "(direct send - no custapi user)"} />
                <Row label="To" value={row.to_address} />
                <Row label="Subject" value={row.subject} />
                <Row label="Attempts" value={`${row.attempts} / ${row.max_attempts}`} />
                <Row label="Scheduled" value={new Date(row.scheduled_at).toLocaleString()} />
                <Row label="Created" value={new Date(row.created_at).toLocaleString()} />
                <Row label="Updated" value={new Date(row.updated_at).toLocaleString()} />
                {row.sent_at && (
                  <Row label="Sent at" value={new Date(row.sent_at).toLocaleString()} />
                )}
                {row.provider && <Row label="Provider" value={row.provider} />}
                {row.provider_message_id && (
                  <Row label="Provider msg id" value={row.provider_message_id} />
                )}
              </Stack>
            </Card>
          </Grid.Col>

          <Grid.Col span={{ base: 12, md: 6 }}>
            {row.last_error && (
              <Alert color="red" title="Last error">
                <Code block>{row.last_error}</Code>
              </Alert>
            )}
            <Card withBorder mt={row.last_error ? "md" : 0}>
              <Text size="sm" c="dimmed" mb="xs">
                Lifecycle hint
              </Text>
              <Text size="sm">
                The worker polls the outbox every few seconds, claims due rows with
                <Code> FOR UPDATE SKIP LOCKED</Code>, dispatches them via the configured sender,
                and marks them <Code>sent</Code>, <Code>failed</Code> (will retry) or
                <Code>dead</Code> (gave up). Refresh this page to see the latest state.
              </Text>
            </Card>
          </Grid.Col>
        </Grid>
      ) : null}
    </Stack>
  );
}

function Row(props: { label: string; value: string }) {
  return (
    <Group justify="space-between" wrap="nowrap">
      <Text size="sm" c="dimmed">
        {props.label}
      </Text>
      <Text size="sm" style={{ wordBreak: "break-all", textAlign: "right" }}>
        {props.value}
      </Text>
    </Group>
  );
}
