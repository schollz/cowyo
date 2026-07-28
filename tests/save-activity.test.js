import assert from "node:assert/strict";
import test from "node:test";

import {
  createSaveActivityIndicator,
  SAVE_ACTIVITY_IDLE_DELAY,
} from "../src/save-activity.js";

test("keeps the cow active until one second after the latest typing", () => {
  const classes = new Set();
  const scheduled = [];
  const cleared = [];
  let nextTimer = 1;
  const element = {
    classList: {
      add(className) {
        classes.add(className);
      },
      remove(className) {
        classes.delete(className);
      },
    },
  };
  const markSaveActivity = createSaveActivityIndicator(element, {
    setTimeout(callback, delay) {
      const timer = nextTimer++;
      scheduled.push({ callback, delay, timer });
      return timer;
    },
    clearTimeout(timer) {
      cleared.push(timer);
    },
  });

  markSaveActivity();
  assert.equal(classes.has("is-save-active"), true);
  assert.equal(scheduled[0].delay, SAVE_ACTIVITY_IDLE_DELAY);

  markSaveActivity();
  assert.deepEqual(cleared, [scheduled[0].timer]);
  assert.equal(scheduled[1].delay, SAVE_ACTIVITY_IDLE_DELAY);
  assert.equal(classes.has("is-save-active"), true);

  scheduled[1].callback();
  assert.equal(classes.has("is-save-active"), false);
});
