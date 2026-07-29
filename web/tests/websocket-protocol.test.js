import assert from "node:assert/strict";
import test from "node:test";

import {
  cursorMessage,
  editMessage,
  operationMessage,
  websocketMessageType,
} from "../src/websocket-protocol.js";

const cursor = { start: 3, end: 6 };

test("builds typed cursor and edit messages", () => {
  assert.deepEqual(cursorMessage(cursor), {
    type: websocketMessageType.cursor,
    cursor_start: 3,
    cursor_end: 6,
  });
  assert.deepEqual(editMessage("shared", cursor), {
    type: websocketMessageType.edit,
    text: "shared",
    cursor_start: 3,
    cursor_end: 6,
  });
});

test("builds typed page operation messages", () => {
  assert.deepEqual(
    operationMessage("lock", "shared", "secret", cursor),
    {
      type: websocketMessageType.operation,
      operation: "lock",
      password: "secret",
      text: "shared",
      cursor_start: 3,
      cursor_end: 6,
    },
  );
});
