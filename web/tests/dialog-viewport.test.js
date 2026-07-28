import assert from "node:assert/strict";
import test from "node:test";

import {
  dialogVisualViewportLayout,
  positionDialogInVisualViewport,
  resetDialogVisualViewportPosition,
} from "../src/dialog-viewport.js";

function testDialog(open = true) {
  const properties = new Map();

  return {
    open,
    properties,
    style: {
      setProperty(name, value) {
        properties.set(name, value);
      },
      removeProperty(name) {
        properties.delete(name);
      },
    },
  };
}

test("moves and constrains the dialog to the visible mobile viewport", () => {
  assert.deepEqual(
    dialogVisualViewportLayout({ height: 360, offsetTop: 12 }),
    {
      center: 192,
      maxHeight: 328,
    },
  );

  const dialog = testDialog();
  assert.equal(
    positionDialogInVisualViewport(dialog, {
      height: 360,
      offsetTop: 12,
    }),
    true,
  );
  assert.equal(
    dialog.properties.get("--crypto-dialog-viewport-center"),
    "192px",
  );
  assert.equal(
    dialog.properties.get("--crypto-dialog-viewport-max-height"),
    "328px",
  );
});

test("does not position a closed dialog and clears positioning on close", () => {
  const dialog = testDialog(false);

  assert.equal(
    positionDialogInVisualViewport(dialog, {
      height: 360,
      offsetTop: 0,
    }),
    false,
  );
  assert.equal(dialog.properties.size, 0);

  dialog.open = true;
  positionDialogInVisualViewport(dialog, {
    height: 360,
    offsetTop: 0,
  });
  resetDialogVisualViewportPosition(dialog);
  assert.equal(dialog.properties.size, 0);
});
