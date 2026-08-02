import assert from "node:assert/strict";
import test from "node:test";

import {
  createPrivateStatusController,
  PRIVATE_ACTIVE_STATUS,
  PRIVATE_KEY_ERROR_STATUS,
} from "../src/private-status.js";

test("dismisses the active private-page notice for the current session", () => {
  const insideTarget = {};
  const container = {
    dataset: {},
    hidden: true,
    contains(target) {
      return target === insideTarget;
    },
  };
  const messageElement = { textContent: "" };
  const status = createPrivateStatusController(container, messageElement);

  status.show(PRIVATE_ACTIVE_STATUS);
  assert.equal(container.hidden, false);
  assert.equal(messageElement.textContent, PRIVATE_ACTIVE_STATUS);
  assert.equal(container.dataset.error, "false");
  assert.equal(container.dataset.active, "true");
  assert.equal(container.dataset.anchored, "true");
  assert.equal(container.dataset.keyError, "false");

  status.dismiss();
  assert.equal(container.hidden, true);

  status.show(PRIVATE_ACTIVE_STATUS);
  assert.equal(container.hidden, true);

  status.show("Complete private URL copied.");
  assert.equal(container.hidden, false);
  assert.equal(messageElement.textContent, "Complete private URL copied.");
  assert.equal(container.dataset.active, "false");
  assert.equal(container.dataset.anchored, "false");
});

test("dismisses a visible private-page notice only for outside presses", () => {
  const insideTarget = {};
  const outsideTarget = {};
  const container = {
    dataset: {},
    hidden: true,
    contains(target) {
      return target === insideTarget;
    },
  };
  const messageElement = { textContent: "" };
  const status = createPrivateStatusController(container, messageElement);

  status.show(PRIVATE_ACTIVE_STATUS);
  assert.equal(status.dismissWhenOutside(insideTarget), false);
  assert.equal(container.hidden, false);

  assert.equal(status.dismissWhenOutside(outsideTarget), true);
  assert.equal(container.hidden, true);
  assert.equal(status.dismissWhenOutside(outsideTarget), false);
});

test("shows the active notice only for an empty private document", () => {
  const container = {
    dataset: {},
    hidden: true,
    contains() {
      return false;
    },
  };
  const messageElement = { textContent: "Authenticating…" };
  const status = createPrivateStatusController(container, messageElement);

  assert.equal(status.showActiveWhenEmpty("already written"), false);
  assert.equal(container.hidden, true);

  assert.equal(status.showActiveWhenEmpty(""), true);
  assert.equal(container.hidden, false);
  assert.equal(messageElement.textContent, PRIVATE_ACTIVE_STATUS);

  assert.equal(status.hideActiveWhenContent("new text"), true);
  assert.equal(container.hidden, true);
  assert.equal(status.showActiveWhenEmpty(""), false);
});

test("marks the malformed-key error as prompt-anchored", () => {
  const container = {
    dataset: {},
    hidden: true,
    contains() {
      return false;
    },
  };
  const messageElement = { textContent: "" };
  const status = createPrivateStatusController(container, messageElement);

  status.show(PRIVATE_KEY_ERROR_STATUS, true);
  assert.equal(container.hidden, false);
  assert.equal(container.dataset.error, "true");
  assert.equal(container.dataset.active, "false");
  assert.equal(container.dataset.anchored, "true");
  assert.equal(container.dataset.keyError, "true");
});
