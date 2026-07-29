export const websocketMessageType = Object.freeze({
  ack: "ack",
  cursor: "cursor",
  cursorLeave: "cursor-leave",
  edit: "edit",
  error: "error",
  operation: "operation",
  update: "update",
});

export function cursorMessage(cursor) {
  return {
    type: websocketMessageType.cursor,
    cursor_start: cursor.start,
    cursor_end: cursor.end,
  };
}

export function editMessage(text, cursor) {
  return {
    type: websocketMessageType.edit,
    text,
    cursor_start: cursor.start,
    cursor_end: cursor.end,
  };
}

export function operationMessage(operation, text, password, cursor) {
  return {
    type: websocketMessageType.operation,
    operation,
    password,
    text,
    cursor_start: cursor.start,
    cursor_end: cursor.end,
  };
}
