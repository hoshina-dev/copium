import { TextInput, type TextInputProps } from "@mantine/core";
import { forwardRef, useImperativeHandle, useRef } from "react";

import { PARAM_DRAG_MIME } from "./ParamChips";

export interface DroppableTextInputHandle {
  insertAtCaret: (snippet: string) => void;
  focus: () => void;
}

interface Props extends Omit<TextInputProps, "value" | "onChange"> {
  value: string;
  onChange: (next: string) => void;
  onFocusChange?: (focused: boolean) => void;
}

// Mirror of DroppableTextarea for the single-line subject input. We can't
// caret-from-point reliably on inputs, so drops always insert at the
// current caret position - good enough since users naturally click into
// the field before dragging.
export const DroppableTextInput = forwardRef<DroppableTextInputHandle, Props>(
  function DroppableTextInput(props, ref) {
    const { value, onChange, onFocusChange, ...rest } = props;
    const inputRef = useRef<HTMLInputElement | null>(null);

    function setValueWithCaret(next: string, caret: number) {
      onChange(next);
      requestAnimationFrame(() => {
        const el = inputRef.current;
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
        const el = inputRef.current;
        const idx = el ? el.selectionStart ?? value.length : value.length;
        insertAtIndex(idx, snippet);
      },
      focus() {
        inputRef.current?.focus();
      },
    }));

    return (
      <TextInput
        {...rest}
        value={value}
        onChange={(e) => onChange(e.currentTarget.value)}
        ref={(el) => {
          inputRef.current = el;
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
          const el = inputRef.current;
          const idx = el?.selectionStart ?? value.length;
          insertAtIndex(idx, `{{.${name}}}`);
        }}
      />
    );
  },
);
