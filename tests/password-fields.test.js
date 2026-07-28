import assert from "node:assert/strict";
import test from "node:test";

import {
  passwordVisibilityState,
  validatePasswordFields,
} from "../src/password-fields.js";

test("accepts matching passwords when confirmation is required", () => {
  assert.equal(
    validatePasswordFields({
      password: "matching password",
      confirmation: "matching password",
      minLength: 8,
    }),
    undefined,
  );
});

test("rejects mismatched password confirmation", () => {
  assert.deepEqual(
    validatePasswordFields({
      password: "matching password",
      confirmation: "different password",
      minLength: 8,
    }),
    {
      field: "confirmation",
      message: "Passwords do not match.",
    },
  );
});

test("does not require confirmation for decrypting or unlocking", () => {
  assert.equal(
    validatePasswordFields({
      password: "p",
      confirmation: undefined,
      minLength: 1,
    }),
    undefined,
  );
});

test("checks password length before confirmation", () => {
  assert.deepEqual(
    validatePasswordFields({
      password: "short",
      confirmation: "short",
      minLength: 8,
    }),
    {
      field: "password",
      message: "Use a password with at least 8 characters.",
    },
  );
});

test("keeps passwords hidden by default", () => {
  assert.deepEqual(passwordVisibilityState(false), {
    inputType: "password",
    label: "Show password",
  });
});

test("reveals passwords with an accessible hide action", () => {
  assert.deepEqual(passwordVisibilityState(true), {
    inputType: "text",
    label: "Hide password",
  });
});
