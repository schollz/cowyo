import assert from "node:assert/strict";
import test from "node:test";

import {
  createPrivateStatusController,
  PRIVATE_ACTIVE_STATUS,
} from "../src/private-status.js";

test("dismisses the active private-page notice for the current session", () => {
  const container = { dataset: {}, hidden: true };
  const messageElement = { textContent: "" };
  const status = createPrivateStatusController(container, messageElement);

  status.show(PRIVATE_ACTIVE_STATUS);
  assert.equal(container.hidden, false);
  assert.equal(messageElement.textContent, PRIVATE_ACTIVE_STATUS);
  assert.equal(container.dataset.error, "false");

  status.dismiss();
  assert.equal(container.hidden, true);

  status.show(PRIVATE_ACTIVE_STATUS);
  assert.equal(container.hidden, true);

  status.show("Complete private URL copied.");
  assert.equal(container.hidden, false);
  assert.equal(messageElement.textContent, "Complete private URL copied.");
});
