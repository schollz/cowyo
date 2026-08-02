import assert from "node:assert/strict";
import test from "node:test";

import {
  base64URLToBytes,
  bytesToBase64URL,
  createLatestResultQueue,
  createSerialQueue,
  decryptE2EEDocument,
  derivePageKeys,
  encodeMasterKey,
  encryptE2EEDocument,
  generateMasterKey,
  parseFragmentKey,
  privatePageURL,
} from "../src/e2ee.js";

const path = "/calm-cat";

test("encodes and strictly parses one 256-bit fragment key", () => {
  const masterKey = generateMasterKey();
  const encoded = encodeMasterKey(masterKey);
  assert.equal(encoded.length, 43);
  assert.deepEqual(parseFragmentKey(`#key=${encoded}`), masterKey);

  for (const fragment of [
    "",
    "#key=",
    `#key=${encoded}=`,
    `#key=${encoded}&other=1`,
    `#other=1&key=${encoded}`,
    `#key=${encoded.slice(1)}`,
  ]) {
    assert.throws(() => parseFragmentKey(fragment), /key/i);
  }
});

test("derives independent page-specific content and write keys", () => {
  const masterKey = Uint8Array.from({ length: 32 }, (_, index) => index);
  const first = derivePageKeys(masterKey, path);
  const again = derivePageKeys(masterKey, path);
  const otherPage = derivePageKeys(masterKey, "/calm-cow");

  assert.deepEqual(first.contentKey, again.contentKey);
  assert.deepEqual(first.writeKey, again.writeKey);
  assert.notDeepEqual(first.contentKey, first.writeKey);
  assert.notDeepEqual(first.contentKey, otherPage.contentKey);
  assert.notDeepEqual(first.writeKey, otherPage.writeKey);
});

test("round trips complete Unicode documents with fresh nonces", () => {
  const { contentKey } = derivePageKeys(generateMasterKey(), path);
  const plaintext = "private notes 🐄\nsecond line";
  const first = encryptE2EEDocument(plaintext, contentKey, path);
  const second = encryptE2EEDocument(plaintext, contentKey, path);

  assert.notEqual(first, second);
  assert.equal(decryptE2EEDocument(first, contentKey, path), plaintext);
  assert.equal(decryptE2EEDocument(second, contentKey, path), plaintext);
});

test("rejects wrong keys, tampering, wrong-page AAD, and malformed envelopes", () => {
  const { contentKey, writeKey } = derivePageKeys(generateMasterKey(), path);
  const document = encryptE2EEDocument("secret", contentKey, path);
  const tampered = document.replace(/"data":"(.)/, (_, character) =>
    `"data":"${character === "A" ? "B" : "A"}`,
  );

  assert.throws(() => decryptE2EEDocument(document, writeKey, path), /wrong key/i);
  assert.throws(() => decryptE2EEDocument(tampered, contentKey, path), /wrong key/i);
  assert.throws(
    () => decryptE2EEDocument(document, contentKey, "/other-page"),
    /wrong key/i,
  );
  assert.throws(
    () => decryptE2EEDocument("not an envelope", contentKey, path),
    /malformed/i,
  );
});

test("builds a clean fragment-only private URL", () => {
  const key = bytesToBase64URL(new Uint8Array(32));
  const location = {
    href: "https://cowyo.example/calm-cat?convert=1#old",
  };
  const url = new URL(privatePageURL(location, key));

  assert.equal(url.pathname, path);
  assert.equal(url.search, "");
  assert.equal(url.hash, `#key=${key}`);
  assert.deepEqual(base64URLToBytes(key, 32), new Uint8Array(32));
});

test("serializes crypto work and suppresses stale queued results", async () => {
  const serial = createSerialQueue();
  const order = [];
  const first = serial.run(async () => {
    await new Promise((resolve) => setTimeout(resolve, 10));
    order.push("first");
  });
  const second = serial.run(async () => {
    order.push("second");
  });
  await Promise.all([first, second]);
  assert.deepEqual(order, ["first", "second"]);

  const applied = [];
  const latest = createLatestResultQueue((value) => applied.push(value));
  const stale = latest.run(async () => "stale");
  const current = latest.run(async () => "current");
  assert.equal((await stale).applied, false);
  assert.equal((await current).applied, true);
  assert.deepEqual(applied, ["current"]);
});
