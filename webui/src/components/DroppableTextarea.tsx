import { Textarea, type TextareaProps } from "@mantine/core";
import { forwardRef, useImperativeHandle, useRef } from "react";

import { PARAM_DRAG_MIME } from "./ParamChips";

// A Mantine Textarea that:
//  - Reports clicks/keyups so the parent knows the latest caret position
//    (so the param chip strip can insert at the right spot when you click
//    a chip after focusing this field).
//  - Accepts dropped param chips - drops insert `{{.name}}` at the
//    drop point, not the previous caret, which feels right.
//
// The ref exposes an `insertAtCaret` method so a parent can drive insertion
// imperatively (eg. from a button click on a chip).

export interface DroppableTextareaHandle {
  insertAtCaret: (snippet: string) => void;
  focus: () => void;
}

interface Props extends Omit<TextareaProps, "value" | "onChange"> {
  value: string;
  onChange: (next: string) => void;
  onFocusChange?: (focused: boolean) => void;
}

export const DroppableTextarea = forwardRef<DroppableTextareaHandle, Props>(function DroppableTextarea(
  props,
  ref,
) {
  const { value, onChange, onFocusChange, ...rest } = props;
  const taRef = useRef<HTMLTextAreaElement | null>(null);

  function setValueWithCaret(next: string, caret: number) {
    onChange(next);
    requestAnimationFrame(() => {
      const el = taRef.current;
      if (!el) return;
      el.focus();
      el.setSelectionRange(caret, caret);
    });
  }

  function insertAtIndex(index: number, snippet: string) {
    const before = value.slice(0, index);
    const after = value.slice(index);
    setValueWithCaret(before + snippet + after, index + snippet.length);
  }

  useImperativeHandle(ref, () => ({
    insertAtCaret(snippet: string) {
      const el = taRef.current;
      const idx = el ? el.selectionStart ?? value.length : value.length;
      insertAtIndex(idx, snippet);
    },
    focus() {
      taRef.current?.focus();
    },
  }));

  return (
    <Textarea
      {...rest}
      value={value}
      onChange={(e) => onChange(e.currentTarget.value)}
      ref={(el) => {
        taRef.current = el;
      }}
      onFocus={() => onFocusChange?.(true)}
      onBlur={() => onFocusChange?.(false)}
      onDragOver={(e) => {
        if (e.dataTransfer.types.includes(PARAM_DRAG_MIME)) {
          e.preventDefault();
          e.dataTransfer.dropEffect = "copy";
        }
      }}
      onDrop={(e) => {
        const name = e.dataTransfer.getData(PARAM_DRAG_MIME);
        if (!name) return;
        e.preventDefault();
        const el = taRef.current;
        if (!el) return;
        // Best-effort caret-at-drop: try the standard caretPositionFromPoint
        // (Firefox) and caretRangeFromPoint (Chromium). If neither works
        // we fall back to the current selection.
        const idx = caretIndexFromEvent(el, e) ?? el.selectionStart ?? value.length;
        insertAtIndex(idx, `{{.${name}}}`);
      }}
    />
  );
});

function caretIndexFromEvent(
  el: HTMLTextAreaElement,
  e: { clientX: number; clientY: number },
): number | null {
  type LegacyDoc = Document & {
    caretPositionFromPoint?: (x: number, y: number) => { offsetNode: Node; offset: number } | null;
    caretRangeFromPoint?: (x: number, y: number) => Range | null;
  };
  const doc = document as LegacyDoc;

  if (typeof doc.caretPositionFromPoint === "function") {
    const pos = doc.caretPositionFromPoint(e.clientX, e.clientY);
    if (pos && pos.offsetNode === el) return pos.offset;
  }
  if (typeof doc.caretRangeFromPoint === "function") {
    const range = doc.caretRangeFromPoint(e.clientX, e.clientY);
    if (range && range.startContainer === el) return range.startOffset;
  }
  // Textareas often refuse caret-from-point because their text isn't a real
  // DOM text node. As a pragmatic fallback we approximate by ratio of the
  // drop Y inside the element.
  const rect = el.getBoundingClientRect();
  const ratio = Math.min(1, Math.max(0, (e.clientY - rect.top) / Math.max(1, rect.height)));
  return Math.round(ratio * el.value.length);
}
