import assert from "node:assert/strict";
import test from "node:test";

import {
  clampRemoteCursorPosition,
  createCursorBroadcastGuard,
  createRemoteCursorOverlay,
  cursorPositionChanged,
  renderRemoteCursorSnapshot,
} from "../src/remote-cursors.js";

function createFakeDocument() {
  const document = {
    createTextNode(text) {
      return { text };
    },
    createElement(tagName) {
      const element = {
        tagName,
        ownerDocument: document,
        parentElement: undefined,
        children: [],
        className: "",
        dataset: {},
        attributes: {},
        scrollTop: 0,
        scrollLeft: 0,
        append(node) {
          node.parentElement = element;
          element.children.push(node);
        },
        remove() {
          if (!element.parentElement) {
            return;
          }
          element.parentElement.children =
            element.parentElement.children.filter(
              (child) => child !== element,
            );
          element.parentElement = undefined;
        },
        replaceChildren() {
          element.children.forEach((child) => {
            child.parentElement = undefined;
          });
          element.children = [];
        },
        setAttribute(name, value) {
          element.attributes[name] = value;
        },
      };
      return element;
    },
  };
  return document;
}

function snapshotContents(snapshot) {
  return snapshot.children.map((node) =>
    node.className === "remote-cursor"
      ? ["cursor", node.dataset.clientId]
      : ["text", node.text],
  );
}

test("clamps remote cursor positions", () => {
  assert.equal(clampRemoteCursorPosition(-5, 6), 0);
  assert.equal(clampRemoteCursorPosition(3.8, 6), 3);
  assert.equal(clampRemoteCursorPosition(99, 6), 6);
  assert.equal(clampRemoteCursorPosition(Number.NaN, 6), undefined);
});

test("renders a gray-line marker at a remote cursor offset", () => {
  const document = createFakeDocument();
  const snapshot = document.createElement("div");
  const textarea = {
    value: "shared note",
    scrollTop: 18,
    scrollLeft: 4,
  };

  renderRemoteCursorSnapshot(
    textarea,
    snapshot,
    "second-editor",
    6,
  );

  assert.deepEqual(snapshotContents(snapshot), [
    ["text", "shared"],
    ["cursor", "second-editor"],
    ["text", " note"],
  ]);
  assert.equal(snapshot.scrollTop, textarea.scrollTop);
  assert.equal(snapshot.scrollLeft, textarea.scrollLeft);
});

test("updates one remote cursor without moving idle collaborators", () => {
  const document = createFakeDocument();
  const overlay = document.createElement("div");
  const textarea = {
    value: "abcdefghij",
    scrollTop: 0,
    scrollLeft: 0,
  };
  const cursors = createRemoteCursorOverlay(textarea, overlay);

  cursors.update("idle-editor", 10);
  const idleSnapshot = overlay.children[0];
  const idleContents = snapshotContents(idleSnapshot);

  textarea.value = "Z\nabcdefghij";
  cursors.update("moving-editor", 12);

  assert.equal(overlay.children[0], idleSnapshot);
  assert.deepEqual(snapshotContents(idleSnapshot), idleContents);
  assert.deepEqual(snapshotContents(overlay.children[1]), [
    ["text", "Z\nabcdefghij"],
    ["cursor", "moving-editor"],
    ["text", ""],
  ]);

  cursors.remove("moving-editor");
  assert.deepEqual(overlay.children, [idleSnapshot]);
  assert.deepEqual(snapshotContents(idleSnapshot), idleContents);
});

test("ignores edit-induced scroll but follows deliberate scrolling", () => {
  const document = createFakeDocument();
  const overlay = document.createElement("div");
  const textarea = {
    value: "shared note",
    scrollTop: 40,
    scrollLeft: 5,
  };
  const cursors = createRemoteCursorOverlay(textarea, overlay);
  cursors.update("idle-editor", 6);
  const snapshot = overlay.children[0];

  cursors.beginTextChange();
  textarea.scrollTop = 80;
  textarea.scrollLeft = 12;
  assert.equal(cursors.syncScroll(), false);
  cursors.finishTextChange();
  assert.equal(cursors.syncScroll(), false);
  assert.deepEqual(
    { top: snapshot.scrollTop, left: snapshot.scrollLeft },
    { top: 40, left: 5 },
  );

  textarea.scrollTop = 90;
  assert.equal(cursors.syncScroll(), true);
  assert.deepEqual(
    { top: snapshot.scrollTop, left: snapshot.scrollLeft },
    { top: 90, left: 12 },
  );
});

test("does not echo programmatic cursor changes as user presence", () => {
  const callbacks = new Map();
  const cleared = [];
  let nextTimer = 1;
  const guard = createCursorBroadcastGuard({
    clearTimeout(timer) {
      if (timer !== undefined) {
        cleared.push(timer);
        callbacks.delete(timer);
      }
    },
    setTimeout(callback, delay) {
      assert.equal(delay, 0);
      const timer = nextTimer++;
      callbacks.set(timer, callback);
      return timer;
    },
  });

  assert.equal(guard.canBroadcast(), true);
  guard.pause();
  assert.equal(guard.canBroadcast(), false);

  guard.resumeAfterCurrentTask();
  assert.equal(guard.canBroadcast(), false);

  guard.pause();
  assert.deepEqual(cleared, [1]);
  assert.equal(guard.canBroadcast(), false);

  guard.resumeAfterCurrentTask();
  callbacks.get(2)();
  assert.equal(guard.canBroadcast(), true);
});

test("broadcasts only genuine cursor position changes", () => {
  const announced = { start: 4, end: 4 };

  assert.equal(
    cursorPositionChanged(announced, { start: 4, end: 4 }),
    false,
  );
  assert.equal(
    cursorPositionChanged(announced, { start: 4, end: 7 }),
    true,
  );
  assert.equal(
    cursorPositionChanged(undefined, { start: 4, end: 4 }),
    true,
  );
});
