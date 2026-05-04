import { Box, Paper } from "@mantine/core";

// Renders rendered HTML in a sandboxed iframe so any inline scripts or
// stylesheets inside the template can't pollute the host UI. We use a
// `srcDoc` blob rather than blob: URLs so the iframe content is always
// in sync with the latest preview.
export function HtmlPreview(props: { html: string; height?: number }) {
  return (
    <Paper withBorder radius="sm" style={{ overflow: "hidden" }}>
      <Box
        component="iframe"
        title="Rendered preview"
        sandbox=""
        srcDoc={props.html || "<html><body><i>(empty preview)</i></body></html>"}
        style={{ width: "100%", height: props.height ?? 480, border: 0, display: "block" }}
      />
    </Paper>
  );
}
