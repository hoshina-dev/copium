import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ActionIcon,
  Badge,
  Button,
  Card,
  Group,
  Loader,
  SegmentedControl,
  Select,
  Stack,
  Table,
  Text,
  TextInput,
  Title,
  Tooltip,
} from "@mantine/core";
import { IconPlayerPause, IconPlayerPlay, IconRefresh, IconSearch } from "@tabler/icons-react";
import { Link, useNavigate } from "react-router-dom";

import { emailsApi, type ListOutboxQuery } from "../api/emails";
import type { OutboxRow, OutboxStatus } from "../api/types";

const STATUS_COLOR: Record<string, string> = {
  queued: "gray",
  sending: "blue",
  sent: "green",
  failed: "yellow",
  dead: "red",
};

const POLL_MS = 3000;

type RangeKey = "15m" | "1h" | "24h" | "today" | "week" | "all";

const RANGES: { value: RangeKey; label: string }[] = [
  { value: "15m", label: "Last 15 min" },
  { value: "1h", label: "Last 1 hour" },
  { value: "24h", label: "Last 24 hours" },
  { value: "today", label: "Today" },
  { value: "week", label: "This week" },
  { value: "all", label: "All time" },
];

function rangeToFrom(r: RangeKey): string | undefined {
  const now = new Date();
  if (r === "all") return undefined;
  if (r === "15m") return new Date(now.getTime() - 15 * 60_000).toISOString();
  if (r === "1h") return new Date(now.getTime() - 60 * 60_000).toISOString();
  if (r === "24h") return new Date(now.getTime() - 24 * 60 * 60_000).toISOString();
  if (r === "today") {
    const d = new Date(now);
    d.setHours(0, 0, 0, 0);
    return d.toISOString();
  }
  // "week": since Monday 00:00 local
  const d = new Date(now);
  const day = (d.getDay() + 6) % 7; // Mon=0
  d.setDate(d.getDate() - day);
  d.setHours(0, 0, 0, 0);
  return d.toISOString();
}

export function OutboxListPage() {
  const navigate = useNavigate();
  const [range, setRange] = useState<RangeKey>("24h");
  const [status, setStatus] = useState<OutboxStatus | "all">("all");
  const [rows, setRows] = useState<OutboxRow[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [polling, setPolling] = useState(true);
  const [lookup, setLookup] = useState("");
  const timerRef = useRef<number | null>(null);

  const load = useCallback(async () => {
    const q: ListOutboxQuery = { limit: 200 };
    const from = rangeToFrom(range);
    if (from) q.from = from;
    if (status !== "all") q.status = status;
    setLoading(true);
    try {
      const list = await emailsApi.listOutbox(q);
      setRows(list);
    } finally {
      setLoading(false);
    }
  }, [range, status]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!polling) return;
    timerRef.current = window.setInterval(() => {
      void load();
    }, POLL_MS);
    return () => {
      if (timerRef.current != null) window.clearInterval(timerRef.current);
    };
  }, [polling, load]);

  const counts = useMemo(() => {
    const c: Record<string, number> = {};
    for (const r of rows ?? []) c[r.status] = (c[r.status] ?? 0) + 1;
    return c;
  }, [rows]);

  return (
    <Stack>
      <Group justify="space-between">
        <Title order={2}>Outbox</Title>
        <Group gap="xs">
          <Tooltip label={polling ? "Pause live updates" : "Resume live updates (every 3s)"}>
            <ActionIcon variant="default" onClick={() => setPolling((p) => !p)}>
              {polling ? <IconPlayerPause size={16} /> : <IconPlayerPlay size={16} />}
            </ActionIcon>
          </Tooltip>
          <Button
            variant="default"
            leftSection={<IconRefresh size={16} />}
            onClick={() => void load()}
            loading={loading}
          >
            Refresh
          </Button>
        </Group>
      </Group>

      <Card withBorder>
        <Stack gap="sm">
          <Group wrap="wrap">
            <SegmentedControl
              value={range}
              onChange={(v) => setRange(v as RangeKey)}
              data={RANGES}
            />
            <Select
              value={status}
              onChange={(v) => setStatus((v as OutboxStatus | "all") ?? "all")}
              data={[
                { value: "all", label: "All statuses" },
                { value: "queued", label: "Queued" },
                { value: "sending", label: "Sending" },
                { value: "sent", label: "Sent" },
                { value: "failed", label: "Failed (retrying)" },
                { value: "dead", label: "Dead (gave up)" },
              ]}
              w={200}
            />
            <Group gap={6}>
              {Object.entries(counts).map(([k, n]) => (
                <Badge key={k} color={STATUS_COLOR[k] ?? "gray"} variant="light">
                  {k}: {n}
                </Badge>
              ))}
            </Group>
            <Group gap={6} style={{ marginLeft: "auto" }}>
              <TextInput
                placeholder="Lookup by outbox UUID"
                value={lookup}
                onChange={(e) => setLookup(e.currentTarget.value)}
                w={320}
              />
              <Button
                leftSection={<IconSearch size={16} />}
                onClick={() => lookup && navigate(`/outbox/${lookup}`)}
                disabled={!lookup}
              >
                Lookup
              </Button>
            </Group>
          </Group>

          <Text size="xs" c="dimmed">
            {polling ? "Live - refreshing every 3s." : "Live updates paused."} Newest first.
            Click a row to see details.
          </Text>
        </Stack>
      </Card>

      {rows === null ? (
        <Loader />
      ) : rows.length === 0 ? (
        <Text c="dimmed">No outbox entries in this range.</Text>
      ) : (
        <Table withTableBorder striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Status</Table.Th>
              <Table.Th>To</Table.Th>
              <Table.Th>Subject</Table.Th>
              <Table.Th>Attempts</Table.Th>
              <Table.Th>Created</Table.Th>
              <Table.Th>Updated</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {rows.map((r) => (
              <Table.Tr
                key={r.id}
                style={{ cursor: "pointer" }}
                onClick={() => navigate(`/outbox/${r.id}`)}
              >
                <Table.Td>
                  <Badge color={STATUS_COLOR[r.status] ?? "gray"}>{r.status}</Badge>
                </Table.Td>
                <Table.Td>{r.to_address}</Table.Td>
                <Table.Td>
                  <Link to={`/outbox/${r.id}`} onClick={(e) => e.stopPropagation()}>
                    {r.subject}
                  </Link>
                </Table.Td>
                <Table.Td>
                  <Text size="sm">
                    {r.attempts} / {r.max_attempts}
                  </Text>
                </Table.Td>
                <Table.Td>
                  <Text size="sm">{new Date(r.created_at).toLocaleString()}</Text>
                </Table.Td>
                <Table.Td>
                  <Text size="sm">{new Date(r.updated_at).toLocaleString()}</Text>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}
    </Stack>
  );
}
