import { useState } from "react";
import { Button, Group, Stack, TextInput, Textarea, Title } from "@mantine/core";
import { useForm } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { useNavigate } from "react-router-dom";

import { templatesApi } from "../api/templates";

export function TemplateNewPage() {
  const navigate = useNavigate();
  const [submitting, setSubmitting] = useState(false);

  const form = useForm({
    initialValues: { code: "", name: "", description: "" },
    validate: {
      code: (v) => (v.trim().length === 0 ? "Required" : null),
      name: (v) => (v.trim().length === 0 ? "Required" : null),
    },
  });

  async function submit(values: typeof form.values) {
    setSubmitting(true);
    try {
      const t = await templatesApi.create(values);
      notifications.show({ color: "green", title: "Template created", message: t.code });
      navigate(`/templates/${t.id}`);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Stack maw={520}>
      <Title order={2}>New template</Title>
      <form onSubmit={form.onSubmit(submit)}>
        <Stack>
          <TextInput
            label="Code"
            description="Stable, machine-friendly identifier (eg. welcome_email)"
            required
            {...form.getInputProps("code")}
          />
          <TextInput
            label="Name"
            description="Human-friendly title shown in the UI"
            required
            {...form.getInputProps("name")}
          />
          <Textarea
            label="Description"
            description="Optional - what is this email for?"
            autosize
            minRows={2}
            {...form.getInputProps("description")}
          />
          <Group justify="flex-end">
            <Button variant="default" onClick={() => navigate(-1)}>
              Cancel
            </Button>
            <Button type="submit" loading={submitting}>
              Create
            </Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  );
}
