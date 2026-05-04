import { Badge, Box, Group, Text, Tooltip } from "@mantine/core";

import type { Param } from "../params/types";

export const PARAM_DRAG_MIME = "application/x-copium-param";

// One row of draggable chips. The parent passes `onInsert(name)` so a click
// inserts at the active editor's caret. Drag uses the standard dataTransfer
// API and the listener side is in DraggableTextarea.
export function ParamChips(props: {
  params: Param[];
  onInsert: (name: string) => void;
  usedNames?: Set<string>;
  hint?: string;
}) {
  const { params, onInsert, usedNames, hint } = props;
  if (params.filter((p) => p.name).length === 0) {
    return null;
  }

  return (
    <Box>
      {hint && (
        <Text size="xs" c="dimmed" mb={4}>
          {hint}
        </Text>
      )}
      <Group gap={6}>
        {params
          .filter((p) => p.name)
          .map((p) => {
            const isUsed = usedNames?.has(p.name);
            const token = `{{.${p.name}}}`;
            return (
              <Tooltip key={p.id} label={`Click or drag to insert ${token}`} withinPortal>
                <Badge
                  variant={isUsed ? "filled" : "light"}
                  color={isUsed ? "blue" : "gray"}
                  size="md"
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.setData(PARAM_DRAG_MIME, p.name);
                    e.dataTransfer.setData("text/plain", token);
                    e.dataTransfer.effectAllowed = "copy";
                  }}
                  onClick={() => onInsert(p.name)}
                  style={{ cursor: "grab", userSelect: "none" }}
                  rightSection={p.required ? "*" : undefined}
                >
                  {p.name}
                </Badge>
              </Tooltip>
            );
          })}
      </Group>
    </Box>
  );
}
