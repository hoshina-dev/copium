import {
  ActionIcon,
  Alert,
  Badge,
  Button,
  Card,
  Group,
  NumberInput,
  Select,
  Stack,
  Switch,
  TagsInput,
  Text,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { IconGripVertical, IconPlus, IconTrash } from "@tabler/icons-react";

import { defaultSampleFor, newParam, type Param, type ParamKind } from "../params/types";

const KIND_OPTIONS: { value: ParamKind; label: string; hint: string }[] = [
  { value: "text", label: "Short text", hint: "First name, city..." },
  { value: "longtext", label: "Long text", hint: "Multi-line copy" },
  { value: "number", label: "Number", hint: "Decimals allowed" },
  { value: "integer", label: "Integer", hint: "Whole number" },
  { value: "boolean", label: "Yes / No", hint: "true or false" },
  { value: "email", label: "Email", hint: "Validates format" },
  { value: "url", label: "URL", hint: "Validates format" },
  { value: "date", label: "Date", hint: "YYYY-MM-DD" },
  { value: "choice", label: "Choice", hint: "Pick from a fixed list" },
];

export interface ParamsBuilderProps {
  params: Param[];
  onChange: (next: Param[]) => void;
  // Names referenced by subject/body so we can show used/unused indicators.
  usedNames: Set<string>;
}

export function ParamsBuilder({ params, onChange, usedNames }: ParamsBuilderProps) {
  function patch(id: string, partial: Partial<Param>) {
    onChange(
      params.map((p) => {
        if (p.id !== id) return p;
        const merged = { ...p, ...partial };
        // When the kind changes, reset the sample so it stays meaningful.
        if (partial.kind && partial.kind !== p.kind) {
          merged.sample = defaultSampleFor(partial.kind);
          if (partial.kind !== "choice") merged.choices = undefined;
        }
        return merged;
      }),
    );
  }

  function remove(id: string) {
    onChange(params.filter((p) => p.id !== id));
  }

  function add() {
    onChange([...params, newParam({ name: nextName(params) })]);
  }

  function move(id: string, dir: -1 | 1) {
    const idx = params.findIndex((p) => p.id === id);
    if (idx < 0) return;
    const target = idx + dir;
    if (target < 0 || target >= params.length) return;
    const copy = [...params];
    [copy[idx], copy[target]] = [copy[target], copy[idx]];
    onChange(copy);
  }

  const nameCounts = new Map<string, number>();
  for (const p of params) {
    if (!p.name) continue;
    nameCounts.set(p.name, (nameCounts.get(p.name) ?? 0) + 1);
  }

  return (
    <Stack gap="sm">
      <Group justify="space-between" align="center">
        <div>
          <Text fw={500}>Variables</Text>
          <Text size="xs" c="dimmed">
            Things that change per send (recipient name, order id, amount…)
          </Text>
        </div>
        <Button size="xs" leftSection={<IconPlus size={14} />} onClick={add}>
          Add variable
        </Button>
      </Group>

      {params.length === 0 && (
        <Alert variant="light" color="gray">
          No variables yet. Click <b>Add variable</b> — then click or drag its chip
          into the subject or body to include it.
        </Alert>
      )}

      {params.map((p, idx) => {
        const dup = nameCounts.get(p.name) ?? 0;
        const nameError =
          dup > 1
            ? "Duplicate name"
            : p.name && !/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(p.name)
            ? "Use letters, digits and _ (no spaces)"
            : undefined;
        const isUsed = usedNames.has(p.name);

        return (
          <Card key={p.id} withBorder padding="xs">
            <Stack gap="xs">
              <Group gap="xs" wrap="nowrap">
                <Tooltip label="Drag to reorder (use the arrows for now)">
                  <ActionIcon variant="subtle" color="gray" disabled>
                    <IconGripVertical size={16} />
                  </ActionIcon>
                </Tooltip>
                <TextInput
                  flex={1}
                  size="xs"
                  placeholder="variable_name (e.g. first_name)"
                  value={p.name}
                  onChange={(e) => patch(p.id, { name: e.currentTarget.value.trim() })}
                  error={nameError}
                />
                <Select
                  size="xs"
                  w={140}
                  data={KIND_OPTIONS.map((k) => ({ value: k.value, label: k.label }))}
                  value={p.kind}
                  onChange={(v) => v && patch(p.id, { kind: v as ParamKind })}
                  allowDeselect={false}
                />
                <Tooltip label="Must be supplied on every send">
                  <Switch
                    size="xs"
                    label="required"
                    checked={p.required}
                    onChange={(e) => patch(p.id, { required: e.currentTarget.checked })}
                  />
                </Tooltip>
                {p.name &&
                  (isUsed ? (
                    <Badge color="green" variant="light" size="xs">
                      used
                    </Badge>
                  ) : (
                    <Badge color="gray" variant="light" size="xs">
                      unused
                    </Badge>
                  ))}
                <Tooltip label="Move up">
                  <ActionIcon
                    variant="subtle"
                    size="sm"
                    onClick={() => move(p.id, -1)}
                    disabled={idx === 0}
                  >
                    ↑
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Move down">
                  <ActionIcon
                    variant="subtle"
                    size="sm"
                    onClick={() => move(p.id, 1)}
                    disabled={idx === params.length - 1}
                  >
                    ↓
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Remove">
                  <ActionIcon variant="subtle" color="red" size="sm" onClick={() => remove(p.id)}>
                    <IconTrash size={14} />
                  </ActionIcon>
                </Tooltip>
              </Group>

              <Group gap="xs" wrap="nowrap">
                <TextInput
                  flex={1}
                  size="xs"
                  placeholder="What is this param for? (shown as helper text)"
                  value={p.description ?? ""}
                  onChange={(e) => patch(p.id, { description: e.currentTarget.value })}
                />
              </Group>

              {/* Sample value editor: type-aware */}
              <SampleEditor param={p} onChange={(v) => patch(p.id, { sample: v })} />

              {p.kind === "choice" && (
                <TagsInput
                  size="xs"
                  label="Allowed values"
                  description="Press Enter after each. The send form will render a dropdown."
                  value={p.choices ?? []}
                  onChange={(v) => patch(p.id, { choices: v })}
                  placeholder="add value"
                />
              )}
            </Stack>
          </Card>
        );
      })}
    </Stack>
  );
}

function SampleEditor(props: { param: Param; onChange: (v: unknown) => void }) {
  const { param, onChange } = props;
  const label = "Sample value";
  const desc = "Used in the live preview. Not persisted with the template.";

  switch (param.kind) {
    case "boolean":
      return (
        <Switch
          size="xs"
          label={label}
          description={desc}
          checked={Boolean(param.sample)}
          onChange={(e) => onChange(e.currentTarget.checked)}
        />
      );
    case "number":
    case "integer":
      return (
        <NumberInput
          size="xs"
          label={label}
          description={desc}
          allowDecimal={param.kind === "number"}
          value={typeof param.sample === "number" ? param.sample : undefined}
          onChange={(v) => onChange(typeof v === "number" ? v : Number(v) || 0)}
        />
      );
    case "longtext":
      return (
        <TextInput
          size="xs"
          label={label}
          description={desc}
          value={typeof param.sample === "string" ? param.sample : ""}
          onChange={(e) => onChange(e.currentTarget.value)}
        />
      );
    case "choice":
      if (!param.choices || param.choices.length === 0) {
        return (
          <Text size="xs" c="dimmed">
            Add at least one allowed value below to set a sample.
          </Text>
        );
      }
      return (
        <Select
          size="xs"
          label={label}
          description={desc}
          data={param.choices.map((c) => ({ value: c, label: c }))}
          value={typeof param.sample === "string" ? param.sample : null}
          onChange={(v) => onChange(v ?? "")}
        />
      );
    default:
      return (
        <TextInput
          size="xs"
          label={label}
          description={desc}
          value={typeof param.sample === "string" ? param.sample : String(param.sample ?? "")}
          onChange={(e) => onChange(e.currentTarget.value)}
        />
      );
  }
}

function nextName(params: Param[]): string {
  const taken = new Set(params.map((p) => p.name));
  for (let i = 1; i < 1000; i++) {
    const candidate = `param${i}`;
    if (!taken.has(candidate)) return candidate;
  }
  return "param";
}
