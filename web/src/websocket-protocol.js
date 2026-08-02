export const websocketMessageType = Object.freeze({
  ack: "ack",
  cursor: "cursor",
  cursorLeave: "cursor-leave",
  edit: "edit",
  error: "error",
  e2eeAuthenticate: "e2ee-authenticate",
  e2eeAuthenticated: "e2ee-authenticated",
  e2eeConvert: "e2ee-convert",
  e2eeCreate: "e2ee-create",
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

export function e2eeBootstrapMessage(type, text, writeCapability, cursor) {
  if (
    type !== websocketMessageType.e2eeCreate &&
    type !== websocketMessageType.e2eeConvert
  ) {
    throw new Error("Unsupported private page bootstrap type.");
  }
  return {
    type,
    text,
    write_capability: writeCapability,
    cursor_start: cursor.start,
    cursor_end: cursor.end,
  };
}

export function e2eeAuthenticateMessage(writeCapability) {
  return {
    type: websocketMessageType.e2eeAuthenticate,
    write_capability: writeCapability,
  };
}

export function websocketURL(location, place) {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${location.host}/ws?place=${encodeURIComponent(place)}`;
}
