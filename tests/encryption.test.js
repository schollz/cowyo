import assert from "node:assert/strict";
import test from "node:test";

import {
  decryptEncryptedBlocks,
  ENCRYPTED_BLOCK_END,
  ENCRYPTED_BLOCK_START,
  encryptText,
  findEncryptedBlocks,
  hasEncryptedBlocks,
} from "../src/encryption.js";

const password = "correct horse battery staple";

test("encrypts and decrypts authenticated Unicode text", async () => {
  const plaintext = "secret cows 🐄\nsecond line";
  const encrypted = await encryptText(plaintext, password);

  assert.ok(encrypted.startsWith(`${ENCRYPTED_BLOCK_START}\n`));
  assert.ok(encrypted.endsWith(`\n${ENCRYPTED_BLOCK_END}`));
  assert.equal(encrypted.includes(plaintext), false);
  assert.equal(hasEncryptedBlocks(encrypted), true);
  assert.equal(findEncryptedBlocks(encrypted).length, 1);

  const decrypted = await decryptEncryptedBlocks(encrypted, password);
  assert.equal(decrypted.text, plaintext);
  assert.equal(decrypted.decryptedCount, 1);
  assert.equal(decrypted.failedCount, 0);
});

test("decrypts only the signed block and preserves surrounding plaintext", async () => {
  const encrypted = await encryptText("inside the block", password);
  const document = `ordinary heading\n${encrypted}\nordinary footer`;

  const decrypted = await decryptEncryptedBlocks(document, password);

  assert.equal(
    decrypted.text,
    "ordinary heading\ninside the block\nordinary footer",
  );
});

test("rejects a wrong password without returning partial plaintext", async () => {
  const encrypted = await encryptText("do not reveal me", password);

  await assert.rejects(
    decryptEncryptedBlocks(encrypted, "this is the wrong password"),
    /Wrong password/,
  );
});

test("detects tampering through the authentication tag", async () => {
  const encrypted = await encryptText("authenticated", password);
  const tampered = encrypted.replace(
    /"data":"([A-Za-z0-9_-])/,
    (_, firstCharacter) => `"data":"${firstCharacter === "A" ? "B" : "A"}`,
  );

  await assert.rejects(
    decryptEncryptedBlocks(tampered, password),
    /encrypted block was changed/,
  );
});

test("requires a meaningful password and a complete signature", async () => {
  await assert.rejects(encryptText("secret", "short"), /at least 8/);
  assert.equal(hasEncryptedBlocks(ENCRYPTED_BLOCK_START), true);

  await assert.rejects(
    decryptEncryptedBlocks(ENCRYPTED_BLOCK_START, password),
    /No complete encrypted block/,
  );
});
