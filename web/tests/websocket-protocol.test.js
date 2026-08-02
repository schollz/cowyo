import assert from "node:assert/strict";
import test from "node:test";

import {
  cursorMessage,
  e2eeAuthenticateMessage,
  e2eeBootstrapMessage,
  editMessage,
  operationMessage,
  websocketURL,
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

test("keeps fragment keys out of WebSocket URLs", () => {
  const location = new URL(
    "https://cowyo.example/calm-cat?private=1#key=never-send-this",
  );
  const url = websocketURL(location, "calm-cat");
  assert.equal(url, "wss://cowyo.example/ws?place=calm-cat");
  assert.doesNotMatch(url, /never-send-this|private=1/);
});

test("builds private bootstrap and authentication messages", () => {
  assert.deepEqual(
    e2eeBootstrapMessage(
      websocketMessageType.e2eeCreate,
      "ciphertext",
      "capability",
      cursor,
    ),
    {
      type: "e2ee-create",
      text: "ciphertext",
      write_capability: "capability",
      cursor_start: 3,
      cursor_end: 6,
    },
  );
  assert.deepEqual(e2eeAuthenticateMessage("capability"), {
    type: "e2ee-authenticate",
    write_capability: "capability",
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
