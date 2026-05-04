import { JsonInput } from "@mantine/core";

// Thin wrapper around Mantine's JsonInput. Centralised so we can swap to a
// fancier code editor (eg. Monaco) later without touching every page.
export function JsonEditor(props: {
  label?: string;
  description?: string;
  value: string;
  onChange: (next: string) => void;
  minRows?: number;
  error?: string;
}) {
  return (
    <JsonInput
      label={props.label}
      description={props.description}
      value={props.value}
      onChange={props.onChange}
      validationError="Invalid JSON"
      formatOnBlur
      autosize
      minRows={props.minRows ?? 8}
      error={props.error}
    />
  );
}
