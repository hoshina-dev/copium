// Bidirectional bridge between the visual Param[] list (UX model) and the
// JSON Schema + sample params that the backend understands. JSON Schema
// stays the source of truth on the wire; this file just gives the UI a way
// to round-trip without forcing users to write JSON.

import { defaultSampleFor, newParam, type Param, type ParamKind } from "./types";

interface JSONSchemaProperty {
  type?: string | string[];
  format?: string;
  enum?: unknown[];
  description?: string;
  default?: unknown;
}

interface JSONSchema {
  type?: "object";
  required?: string[];
  properties?: Record<string, JSONSchemaProperty>;
  // If we round-tripped from UI we set this to anchor sample values too:
  examples?: Array<Record<string, unknown>>;
}

const KIND_TO_SCHEMA: Record<ParamKind, JSONSchemaProperty> = {
  text: { type: "string" },
  longtext: { type: "string", format: "long-text" },
  number: { type: "number" },
  integer: { type: "integer" },
  boolean: { type: "boolean" },
  email: { type: "string", format: "email" },
  url: { type: "string", format: "uri" },
  date: { type: "string", format: "date" },
  choice: { type: "string" }, // enum is added separately
};

function inferKind(p: JSONSchemaProperty): ParamKind {
  if (p.enum && Array.isArray(p.enum) && p.enum.length > 0) return "choice";
  const t = Array.isArray(p.type) ? p.type.find((x) => x !== "null") ?? p.type[0] : p.type;
  switch (t) {
    case "boolean":
      return "boolean";
    case "integer":
      return "integer";
    case "number":
      return "number";
    default:
      switch (p.format) {
        case "email":
          return "email";
        case "uri":
        case "url":
          return "url";
        case "date":
          return "date";
        case "long-text":
          return "longtext";
        default:
          return "text";
      }
  }
}

export function paramsToSchema(params: Param[]): {
  schema: Record<string, unknown>;
  sample: Record<string, unknown>;
} {
  const properties: Record<string, JSONSchemaProperty> = {};
  const required: string[] = [];
  const sample: Record<string, unknown> = {};

  for (const p of params) {
    if (!p.name.trim()) continue;
    const base = { ...KIND_TO_SCHEMA[p.kind] };
    if (p.description) base.description = p.description;
    if (p.kind === "choice" && p.choices && p.choices.length > 0) {
      base.enum = p.choices;
    }
    properties[p.name] = base;
    if (p.required) required.push(p.name);
    sample[p.name] = p.sample === undefined ? defaultSampleFor(p.kind) : p.sample;
  }

  const schema: Record<string, unknown> = { type: "object", properties };
  if (required.length > 0) schema.required = required;
  return { schema, sample };
}

// Best-effort: turn an existing JSON Schema (from a saved version) back into
// the visual Param list so the editor can be reused for "view" mode.
export function schemaToParams(
  schema: Record<string, unknown> | undefined,
  knownSample?: Record<string, unknown>,
): Param[] {
  const s = (schema ?? {}) as JSONSchema;
  const properties = s.properties ?? {};
  const required = new Set(s.required ?? []);
  const out: Param[] = [];
  for (const [name, prop] of Object.entries(properties)) {
    const kind = inferKind(prop);
    const choices =
      kind === "choice" && Array.isArray(prop.enum)
        ? prop.enum.map((v) => String(v))
        : undefined;
    out.push(
      newParam({
        name,
        kind,
        required: required.has(name),
        description: prop.description,
        sample:
          knownSample && name in knownSample
            ? knownSample[name]
            : prop.default !== undefined
            ? prop.default
            : defaultSampleFor(kind),
        choices,
      }),
    );
  }
  return out;
}
