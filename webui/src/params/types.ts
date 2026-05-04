// A "Param" is the visual, user-friendly representation of one template
// variable. It compiles down to a JSON Schema property + a sample value.
//
// Keeping this model UI-only means we can evolve the editor (add/remove
// types, change samples) without breaking the wire contract: every send
// still validates against the JSON Schema we generate from this list.

export type ParamKind =
  | "text"
  | "longtext"
  | "number"
  | "integer"
  | "boolean"
  | "email"
  | "url"
  | "date"
  | "choice";

export interface Param {
  id: string;
  name: string;
  kind: ParamKind;
  required: boolean;
  description?: string;
  sample?: unknown;
  // For "choice" only: the allowed values.
  choices?: string[];
}

// Reasonable starter sample for a freshly added param of a given kind.
// Why non-empty strings: the live preview substitutes samples into the
// template, so an empty default makes `Hi {{.name}}!` render as `Hi !`
// and users reasonably assume the preview is broken. Keep these obviously
// placeholder-looking so nobody mistakes them for real data.
export function defaultSampleFor(kind: ParamKind): unknown {
  switch (kind) {
    case "text":
      return "sample text";
    case "longtext":
      return "sample long-form text";
    case "number":
      return 42;
    case "integer":
      return 42;
    case "boolean":
      return false;
    case "email":
      return "user@example.com";
    case "url":
      return "https://example.com";
    case "date":
      return new Date().toISOString().slice(0, 10);
    case "choice":
      return "sample";
  }
}

export function newParam(seed: Partial<Param> = {}): Param {
  const kind: ParamKind = seed.kind ?? "text";
  return {
    id: cryptoRandomId(),
    name: seed.name ?? "",
    kind,
    required: seed.required ?? false,
    description: seed.description,
    sample: seed.sample ?? defaultSampleFor(kind),
    choices: seed.choices,
  };
}

function cryptoRandomId(): string {
  // crypto.randomUUID exists in modern browsers we target, but fall back to
  // a simple base36 random string just in case (eg. test envs).
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2, 10);
}
