import assert from "node:assert/strict";
import test from "node:test";

import { calculateAnchoredStatusPosition } from "../src/private-status-position.js";

test("places a key error below a final line when space is available", () => {
  assert.deepEqual(
    calculateAnchoredStatusPosition({
      anchorLeft: 180,
      anchorTop: 70,
      anchorBottom: 90,
      statusWidth: 300,
      statusHeight: 80,
      viewportWidth: 800,
      viewportHeight: 600,
    }),
    { left: 156, placement: "below", pointerLeft: 17, top: 100 },
  );
});

test("places a key error above a final line near the viewport bottom", () => {
  assert.deepEqual(
    calculateAnchoredStatusPosition({
      anchorLeft: 780,
      anchorTop: 540,
      anchorBottom: 560,
      statusWidth: 320,
      statusHeight: 100,
      viewportWidth: 800,
      viewportHeight: 600,
    }),
    { left: 464, placement: "above", pointerLeft: 294, top: 430 },
  );
});
