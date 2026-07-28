import assert from "node:assert/strict";
import test from "node:test";

import {
  applySystemTheme,
  applyTheme,
  readStoredTheme,
  storeTheme,
  systemTheme,
  THEME_STORAGE_KEY,
} from "../src/theme.js";

function storageWith(value) {
  return {
    value,
    getItem(key) {
      assert.equal(key, THEME_STORAGE_KEY);
      return this.value;
    },
    setItem(key, nextValue) {
      assert.equal(key, THEME_STORAGE_KEY);
      this.value = nextValue;
    },
  };
}

function themeElements() {
  return {
    root: { dataset: {} },
    meta: {
      content: "",
      setAttribute(name, value) {
        assert.equal(name, "content");
        this.content = value;
      },
    },
  };
}

test("reads only valid cached theme preferences", () => {
  assert.equal(readStoredTheme(storageWith("dark")), "dark");
  assert.equal(readStoredTheme(storageWith("light")), "light");
  assert.equal(readStoredTheme(storageWith("sepia")), undefined);
});

test("uses the system color scheme when there is no override", () => {
  assert.equal(systemTheme({ matches: true }), "dark");
  assert.equal(systemTheme({ matches: false }), "light");

  const { root, meta } = themeElements();
  root.dataset.theme = "light";
  assert.equal(applySystemTheme(root, meta, { matches: true }), "dark");
  assert.equal(root.dataset.theme, undefined);
  assert.equal(meta.content, "#171717");
});

test("applies and stores an explicit site-wide override", () => {
  const storage = storageWith(undefined);
  const { root, meta } = themeElements();

  assert.equal(storeTheme(storage, "dark"), true);
  applyTheme(root, meta, readStoredTheme(storage));

  assert.equal(storage.value, "dark");
  assert.equal(root.dataset.theme, "dark");
  assert.equal(meta.content, "#171717");
});

test("continues applying a theme when local storage is unavailable", () => {
  const unavailableStorage = {
    getItem() {
      throw new Error("blocked");
    },
    setItem() {
      throw new Error("blocked");
    },
  };
  const { root, meta } = themeElements();

  assert.equal(readStoredTheme(unavailableStorage), undefined);
  assert.equal(storeTheme(unavailableStorage, "light"), false);
  applyTheme(root, meta, "light");
  assert.equal(root.dataset.theme, "light");
});
