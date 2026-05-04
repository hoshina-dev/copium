import { useEffect, useState } from "react";
import {
  Badge,
  Button,
  Card,
  Code,
  Group,
  Loader,
  Menu,
  Stack,
  Table,
  Text,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconChevronDown, IconCopy, IconPlus, IconRocket } from "@tabler/icons-react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { templatesApi } from "../api/templates";
import type { Template, TemplateVersion } from "../api/types";

function NewVersionMenu({
  templateId,
  versions,
}: {
  templateId: string;
  versions: TemplateVersion[];
}) {
  const sorted = [...versions].sort((a, b) => b.version - a.version);
  const latest = sorted[0];
  const blank = `/templates/${templateId}/versions/new`;
  const latestHref = latest ? `${blank}/from/${latest.version}` : blank;

  if (!latest) {
    return (
      <Button component={Link} to={blank} leftSection={<IconPlus size={16} />}>
        New version
      </Button>
    );
  }

  return (
    <Menu shadow="md" width={260} position="bottom-end">
      <Menu.Target>
        <Button
          leftSection={<IconPlus size={16} />}
          rightSection={<IconChevronDown size={14} />}
          component={Link}
          // Default click uses the latest version as the base (user's
          // requested default). The dropdown lets them pick any prior
          // version or start from a blank template.
          to={latestHref}
        >
          New version
        </Button>
      </Menu.Target>
      <Menu.Dropdown>
        <Menu.Label>Base new version on…</Menu.Label>
        {sorted.map((v, i) => (
          <Menu.Item
            key={v.id}
            component={Link}
            to={`${blank}/from/${v.version}`}
            leftSection={<IconCopy size={14} />}
            rightSection={i === 0 ? <Badge size="xs">latest</Badge> : null}
          >
            v{v.version} — {v.subject || "(no subject)"}
          </Menu.Item>
        ))}
        <Menu.Divider />
        <Menu.Item component={Link} to={blank}>
          Start from a blank template
        </Menu.Item>
      </Menu.Dropdown>
    </Menu>
  );
}

export function TemplateDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [template, setTemplate] = useState<Template | null>(null);
  const [versions, setVersions] = useState<TemplateVersion[] | null>(null);
  const [pending, setPending] = useState(false);

  async function load() {
    const [t, vs] = await Promise.all([templatesApi.get(id), templatesApi.listVersions(id)]);
    setTemplate(t);
    setVersions(vs);
  }

  useEffect(() => {
    void load();
  }, [id]);

  async function setActive(versionId: string) {
    setPending(true);
    try {
      await templatesApi.setActive(id, versionId);
      notifications.show({ color: "green", title: "Active version updated", message: versionId });
      await load();
    } finally {
      setPending(false);
    }
  }

  if (!template || !versions) {
    return <Loader />;
  }

  return (
    <Stack>
      <Group justify="space-between">
        <div>
          <Title order={2}>{template.name}</Title>
          <Text c="dimmed">
            <Code>{template.code}</Code>
          </Text>
        </div>
        <Group>
          <Button
            leftSection={<IconRocket size={16} />}
            variant="default"
            onClick={() => navigate(`/dispatch/${template.id}`)}
            disabled={!template.active_version_id}
            title={template.active_version_id ? "" : "Set an active version first"}
          >
            Test send
          </Button>
          <NewVersionMenu templateId={template.id} versions={versions} />
        </Group>
      </Group>

      {template.description && (
        <Card withBorder>
          <Text>{template.description}</Text>
        </Card>
      )}

      <Title order={4}>Versions</Title>
      {versions.length === 0 ? (
        <Text c="dimmed">No versions yet. Create one to be able to send emails.</Text>
      ) : (
        <Table withTableBorder striped>
          <Table.Thead>
            <Table.Tr>
              <Table.Th w={70}>#</Table.Th>
              <Table.Th>Subject</Table.Th>
              <Table.Th>From</Table.Th>
              <Table.Th>Created</Table.Th>
              <Table.Th>State</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {versions.map((v) => {
              const isActive = template.active_version_id === v.id;
              return (
                <Table.Tr key={v.id}>
                  <Table.Td>v{v.version}</Table.Td>
                  <Table.Td>
                    <Link to={`/templates/${template.id}/versions/${v.version}`}>{v.subject}</Link>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" c="dimmed">
                      {v.from_address || "(default)"}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm">{new Date(v.created_at).toLocaleString()}</Text>
                  </Table.Td>
                  <Table.Td>
                    {isActive ? (
                      <Badge color="green">active</Badge>
                    ) : (
                      <Badge color="gray" variant="light">
                        draft
                      </Badge>
                    )}
                  </Table.Td>
                  <Table.Td>
                    {!isActive && (
                      <Button
                        size="xs"
                        variant="light"
                        loading={pending}
                        onClick={() => setActive(v.id)}
                      >
                        Set active
                      </Button>
                    )}
                  </Table.Td>
                </Table.Tr>
              );
            })}
          </Table.Tbody>
        </Table>
      )}
    </Stack>
  );
}
