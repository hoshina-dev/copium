import { Alert, NumberInput, Select, Stack, Switch, Textarea, TextInput } from "@mantine/core";

import type { JSONObject, JSONValue } from "../api/types";

// A deliberately tiny schema-driven form: we only inspect the top-level
// `properties` block of a JSON Schema and render an input per property.
// Unsupported shapes (nested objects, arrays, oneOf, ...) fall back to a
// JSON textarea so power users can still send arbitrary payloads.
type Property = {
  type?: string | string[];
  enum?: JSONValue[];
  description?: string;
  format?: string;
  default?: JSONValue;
};

type Schema = {
  type?: string;
  required?: string[];
  properties?: Record<string, Property>;
};

function asProperty(v: unknown): Property {
  return (v ?? {}) as Property;
}

function primaryType(p: Property): string {
  if (Array.isArray(p.type)) {
    const first = p.type.find((t) => t !== "null") ?? p.type[0];
    return String(first ?? "string");
  }
  return String(p.type ?? "string");
}

export function SchemaForm(props: {
  schema: JSONObject | undefined;
  value: JSONObject;
  onChange: (next: JSONObject) => void;
}) {
  const schema = (props.schema ?? {}) as Schema;
  const required = new Set(schema.required ?? []);
  const properties = schema.properties ?? {};
  const keys = Object.keys(properties);

  if (keys.length === 0) {
    return (
      <Alert variant="light" color="gray" title="No schema fields">
        This template has no parameters - press Send to fire it as-is.
      </Alert>
    );
  }

  function setField(key: string, v: JSONValue) {
    props.onChange({ ...props.value, [key]: v });
  }

  return (
    <Stack gap="sm">
      {keys.map((key) => {
        const prop = asProperty(properties[key]);
        const isRequired = required.has(key);
        const label = `${key}${isRequired ? " *" : ""}`;
        const desc = prop.description;
        const current = props.value[key];

        if (prop.enum && Array.isArray(prop.enum)) {
          return (
            <Select
              key={key}
              label={label}
              description={desc}
              data={prop.enum.map((v) => ({ value: String(v), label: String(v) }))}
              value={current === undefined || current === null ? null : String(current)}
              onChange={(v) => setField(key, v)}
            />
          );
        }

        switch (primaryType(prop)) {
          case "boolean":
            return (
              <Switch
                key={key}
                label={label}
                description={desc}
                checked={Boolean(current)}
                onChange={(e) => setField(key, e.currentTarget.checked)}
              />
            );
          case "number":
          case "integer":
            return (
              <NumberInput
                key={key}
                label={label}
                description={desc}
                value={typeof current === "number" ? current : undefined}
                onChange={(v) => setField(key, typeof v === "number" ? v : Number(v) || 0)}
                allowDecimal={primaryType(prop) === "number"}
              />
            );
          default: {
            const isLong = (desc ?? "").length > 60;
            const Component = isLong ? Textarea : TextInput;
            return (
              <Component
                key={key}
                label={label}
                description={desc}
                value={typeof current === "string" ? current : current == null ? "" : String(current)}
                onChange={(e) => setField(key, e.currentTarget.value)}
                autosize={isLong}
                minRows={isLong ? 2 : undefined}
              />
            );
          }
        }
      })}
    </Stack>
  );
}
