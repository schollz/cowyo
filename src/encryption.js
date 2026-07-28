import { xchacha20poly1305 } from "@noble/ciphers/chacha.js";
import { randomBytes } from "@noble/ciphers/utils.js";
import { scryptAsync } from "@noble/hashes/scrypt.js";

export const ENCRYPTED_BLOCK_START =
  "-----BEGIN COWYO ENCRYPTED BLOCK V1-----";
export const ENCRYPTED_BLOCK_END =
  "-----END COWYO ENCRYPTED BLOCK V1-----";

const KDF_OPTIONS = Object.freeze({
  N: 2 ** 16,
  r: 8,
  p: 1,
  dkLen: 32,
});
const SALT_LENGTH = 16;
const NONCE_LENGTH = 24;
const AUTH_TAG_LENGTH = 16;
const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder("utf-8", { fatal: true });
const additionalData = textEncoder.encode(ENCRYPTED_BLOCK_START);

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function encryptedBlockPattern() {
  return new RegExp(
    `${escapeRegExp(ENCRYPTED_BLOCK_START)}\\r?\\n([^\\r\\n]+)\\r?\\n${escapeRegExp(ENCRYPTED_BLOCK_END)}`,
    "g",
  );
}

function bytesToBase64URL(bytes) {
  let binary = "";
  const chunkSize = 0x8000;

  for (let offset = 0; offset < bytes.length; offset += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(offset, offset + chunkSize));
  }

  return btoa(binary)
    .replaceAll("+", "-")
    .replaceAll("/", "_")
    .replace(/=+$/, "");
}

function base64URLToBytes(value) {
  if (typeof value !== "string" || !/^[A-Za-z0-9_-]+$/.test(value)) {
    throw new Error("Encrypted block contains invalid data.");
  }

  const padding = "=".repeat((4 - (value.length % 4)) % 4);
  const binary = atob(
    value.replaceAll("-", "+").replaceAll("_", "/") + padding,
  );
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

function parseEnvelope(payload) {
  let envelope;

  try {
    envelope = JSON.parse(payload);
  } catch {
    throw new Error("Encrypted block is not valid JSON.");
  }

  if (
    envelope?.v !== 1 ||
    envelope?.kdf !== "scrypt" ||
    envelope?.N !== KDF_OPTIONS.N ||
    envelope?.r !== KDF_OPTIONS.r ||
    envelope?.p !== KDF_OPTIONS.p ||
    envelope?.cipher !== "xchacha20-poly1305"
  ) {
    throw new Error("Encrypted block uses an unsupported format.");
  }

  const salt = base64URLToBytes(envelope.salt);
  const nonce = base64URLToBytes(envelope.nonce);
  const ciphertext = base64URLToBytes(envelope.data);

  if (
    salt.length !== SALT_LENGTH ||
    nonce.length !== NONCE_LENGTH ||
    ciphertext.length < AUTH_TAG_LENGTH
  ) {
    throw new Error("Encrypted block contains invalid data.");
  }

  return { salt, nonce, ciphertext };
}

async function decryptPayload(payload, password) {
  const { salt, nonce, ciphertext } = parseEnvelope(payload);
  const key = await scryptAsync(password, salt, KDF_OPTIONS);

  try {
    const plaintext = xchacha20poly1305(
      key,
      nonce,
      additionalData,
    ).decrypt(ciphertext);
    return textDecoder.decode(plaintext);
  } catch {
    throw new Error("Wrong password, or the encrypted block was changed.");
  } finally {
    key.fill(0);
  }
}

export function findEncryptedBlocks(text) {
  return Array.from(text.matchAll(encryptedBlockPattern()), (match) => ({
    start: match.index,
    end: match.index + match[0].length,
    raw: match[0],
    payload: match[1],
  }));
}

export function hasEncryptedBlocks(text) {
  return text.includes(ENCRYPTED_BLOCK_START);
}

export async function encryptText(text, password) {
  if (password.length < 8) {
    throw new Error("Use a password with at least 8 characters.");
  }

  const salt = randomBytes(SALT_LENGTH);
  const nonce = randomBytes(NONCE_LENGTH);
  const key = await scryptAsync(password, salt, KDF_OPTIONS);

  try {
    const ciphertext = xchacha20poly1305(
      key,
      nonce,
      additionalData,
    ).encrypt(textEncoder.encode(text));
    const payload = JSON.stringify({
      v: 1,
      kdf: "scrypt",
      N: KDF_OPTIONS.N,
      r: KDF_OPTIONS.r,
      p: KDF_OPTIONS.p,
      salt: bytesToBase64URL(salt),
      cipher: "xchacha20-poly1305",
      nonce: bytesToBase64URL(nonce),
      data: bytesToBase64URL(ciphertext),
    });

    return `${ENCRYPTED_BLOCK_START}\n${payload}\n${ENCRYPTED_BLOCK_END}`;
  } finally {
    key.fill(0);
  }
}

export async function decryptEncryptedBlocks(text, password) {
  const blocks = findEncryptedBlocks(text);

  if (blocks.length === 0) {
    throw new Error("No complete encrypted block was found.");
  }

  let output = "";
  let position = 0;
  let decryptedCount = 0;
  let failedCount = 0;
  let firstError;

  for (const block of blocks) {
    output += text.slice(position, block.start);

    try {
      output += await decryptPayload(block.payload, password);
      decryptedCount++;
    } catch (error) {
      output += block.raw;
      failedCount++;
      firstError ??= error;
    }

    position = block.end;
  }

  if (decryptedCount === 0) {
    throw firstError;
  }

  output += text.slice(position);
  return { text: output, decryptedCount, failedCount };
}
