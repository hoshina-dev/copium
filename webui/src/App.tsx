import { AppShell, Burger, Group, NavLink, Title, Anchor, Badge } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconTemplate, IconSend, IconInbox, IconBook } from "@tabler/icons-react";
import { Routes, Route, NavLink as RouterNavLink, Navigate, useLocation } from "react-router-dom";

import { TemplatesListPage } from "./pages/TemplatesList";
import { TemplateNewPage } from "./pages/TemplateNew";
import { TemplateDetailPage } from "./pages/TemplateDetail";
import { VersionEditorPage } from "./pages/VersionEditor";
import { DispatchPage } from "./pages/Dispatch";
import { OutboxDetailPage } from "./pages/OutboxDetail";

const NAV = [
  { to: "/templates", label: "Templates", icon: IconTemplate },
  { to: "/dispatch", label: "Dispatch", icon: IconSend },
  { to: "/outbox", label: "Outbox", icon: IconInbox },
];

export function App() {
  const [opened, { toggle }] = useDisclosure();
  const location = useLocation();

  return (
    <AppShell
      header={{ height: 56 }}
      navbar={{ width: 220, breakpoint: "sm", collapsed: { mobile: !opened } }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Group>
            <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
            <Title order={4}>Copium</Title>
            <Badge variant="light" color="gray">
              Internal · No auth
            </Badge>
          </Group>
          <Anchor href="/swagger/index.html" target="_blank" size="sm">
            <Group gap={4}>
              <IconBook size={16} />
              API docs
            </Group>
          </Anchor>
        </Group>
      </AppShell.Header>
      <AppShell.Navbar p="xs">
        {NAV.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.to}
              component={RouterNavLink}
              to={item.to}
              label={item.label}
              leftSection={<Icon size={18} />}
              active={location.pathname.startsWith(item.to)}
            />
          );
        })}
      </AppShell.Navbar>
      <AppShell.Main>
        <Routes>
          <Route path="/" element={<Navigate to="/templates" replace />} />
          <Route path="/templates" element={<TemplatesListPage />} />
          <Route path="/templates/new" element={<TemplateNewPage />} />
          <Route path="/templates/:id" element={<TemplateDetailPage />} />
          <Route path="/templates/:id/versions/new" element={<VersionEditorPage mode="new" />} />
          <Route path="/templates/:id/versions/:version" element={<VersionEditorPage mode="view" />} />
          <Route path="/dispatch" element={<DispatchPage />} />
          <Route path="/dispatch/:templateId" element={<DispatchPage />} />
          <Route path="/outbox" element={<OutboxDetailPage />} />
          <Route path="/outbox/:id" element={<OutboxDetailPage />} />
          <Route path="*" element={<Navigate to="/templates" replace />} />
        </Routes>
      </AppShell.Main>
    </AppShell>
  );
}
