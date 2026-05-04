import { useEffect, useState } from "react";
import { Badge, Button, Group, Loader, Stack, Table, Text, Title } from "@mantine/core";
import { IconPlus, IconRefresh } from "@tabler/icons-react";
import { Link } from "react-router-dom";

import { templatesApi } from "../api/templates";
import type { Template } from "../api/types";

export function TemplatesListPage() {
  const [items, setItems] = useState<Template[] | null>(null);
  const [loading, setLoading] = useState(false);

  async function load() {
    setLoading(true);
    try {
      setItems(await templatesApi.list());
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <Stack>
      <Group justify="space-between">
        <Title order={2}>Templates</Title>
        <Group>
          <Button leftSection={<IconRefresh size={16} />} variant="default" onClick={load} loading={loading}>
            Refresh
          </Button>
          <Button leftSection={<IconPlus size={16} />} component={Link} to="/templates/new">
            New template
          </Button>
        </Group>
      </Group>

      {items === null ? (
        <Loader />
      ) : items.length === 0 ? (
        <Text c="dimmed">No templates yet. Create one to get started.</Text>
      ) : (
        <Table striped highlightOnHover withTableBorder withColumnBorders>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Code</Table.Th>
              <Table.Th>Name</Table.Th>
              <Table.Th>Description</Table.Th>
              <Table.Th>Active version</Table.Th>
              <Table.Th>Updated</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {items.map((t) => (
              <Table.Tr key={t.id}>
                <Table.Td>
                  <Link to={`/templates/${t.id}`}>{t.code}</Link>
                </Table.Td>
                <Table.Td>{t.name}</Table.Td>
                <Table.Td>
                  <Text c="dimmed" size="sm">
                    {t.description || "-"}
                  </Text>
                </Table.Td>
                <Table.Td>
                  {t.active_version_id ? (
                    <Badge color="green" variant="light">
                      active
                    </Badge>
                  ) : (
                    <Badge color="gray" variant="light">
                      no version
                    </Badge>
                  )}
                </Table.Td>
                <Table.Td>
                  <Text size="sm">{new Date(t.updated_at).toLocaleString()}</Text>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}
    </Stack>
  );
}
