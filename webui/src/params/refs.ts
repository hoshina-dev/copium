// Extract `{{.field}}` references from arbitrary template text. Used to
// power the used/unused chip indicators and to warn about undefined refs.
const REF_RE = /\{\{\s*\.(\w+)\s*\}\}/g;

export function collectRefs(...texts: string[]): Set<string> {
  const out = new Set<string>();
  for (const text of texts) {
    if (!text) continue;
    REF_RE.lastIndex = 0;
    let m: RegExpExecArray | null;
    while ((m = REF_RE.exec(text)) !== null) {
      out.add(m[1]);
    }
  }
  return out;
}
