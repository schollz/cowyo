import { xchacha20poly1305 } from "@noble/ciphers/chacha.js";
import { randomBytes } from "@noble/ciphers/utils.js";
import { hkdf } from "@noble/hashes/hkdf.js";
import { sha256 } from "@noble/hashes/sha2.js";

export const E2EE_DOCUMENT_START =
  "-----BEGIN COWYO E2EE DOCUMENT V1-----";
export const E2EE_DOCUMENT_END = "-----END COWYO E2EE DOCUMENT V1-----";

const FORMAT_LABEL = "COWYO E2EE DOCUMENT V1";
const SALT_LABEL = "COWYO E2EE PAGE SALT V1";
const CONTENT_KEY_LABEL = "COWYO E2EE CONTENT KEY V1";
const WRITE_KEY_LABEL = "COWYO E2EE WRITE CAPABILITY V1";
const KEY_LENGTH = 32;
const NONCE_LENGTH = 24;
const AUTH_TAG_LENGTH = 16;
const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder("utf-8", { fatal: true });

export function bytesToBase64URL(bytes) {
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

export function base64URLToBytes(value, expectedLength) {
  if (
    typeof value !== "string" ||
    value.length === 0 ||
    value.includes("=") ||
    !/^[A-Za-z0-9_-]+$/.test(value)
  ) {
    throw new Error("Invalid base64url value.");
  }

  const padding = "=".repeat((4 - (value.length % 4)) % 4);
  let binary;
  try {
    binary = atob(
      value.replaceAll("-", "+").replaceAll("_", "/") + padding,
    );
  } catch {
    throw new Error("Invalid base64url value.");
  }
  const decoded = Uint8Array.from(binary, (character) =>
    character.charCodeAt(0),
  );
  if (
    bytesToBase64URL(decoded) !== value ||
    (expectedLength !== undefined && decoded.length !== expectedLength)
  ) {
    throw new Error("Invalid base64url value.");
  }
  return decoded;
}

export function generateMasterKey() {
  return randomBytes(KEY_LENGTH);
}

export function encodeMasterKey(masterKey) {
  if (!(masterKey instanceof Uint8Array) || masterKey.length !== KEY_LENGTH) {
    throw new Error("Private page keys must contain exactly 32 bytes.");
  }
  return bytesToBase64URL(masterKey);
}

export function parseFragmentKey(fragment) {
  if (typeof fragment !== "string") {
    throw new Error("This private link has a malformed key.");
  }
  const match = /^#key=([A-Za-z0-9_-]{43})$/.exec(fragment);
  if (!match) {
    throw new Error("This private link has a missing or malformed key.");
  }
  try {
    return base64URLToBytes(match[1], KEY_LENGTH);
  } catch {
    throw new Error("This private link has a malformed key.");
  }
}

export function normalizePagePath(pathname) {
  if (typeof pathname !== "string" || pathname === "") {
    throw new Error("A page path is required for private encryption.");
  }
  let decoded;
  try {
    decoded = decodeURIComponent(pathname);
  } catch {
    throw new Error("The page path is malformed.");
  }
  decoded = decoded.normalize("NFC").replace(/^\/+/, "");
  if (decoded === "" || decoded.endsWith("/")) {
    throw new Error("The page path is malformed.");
  }
  return `/${decoded}`;
}

export function derivePageKeys(masterKey, pathname) {
  if (!(masterKey instanceof Uint8Array) || masterKey.length !== KEY_LENGTH) {
    throw new Error("Private page keys must contain exactly 32 bytes.");
  }
  const normalizedPath = normalizePagePath(pathname);
  const salt = sha256(
    textEncoder.encode(`${SALT_LABEL}\0${normalizedPath}`),
  );
  return {
    contentKey: hkdf(
      sha256,
      masterKey,
      salt,
      textEncoder.encode(CONTENT_KEY_LABEL),
      KEY_LENGTH,
    ),
    writeKey: hkdf(
      sha256,
      masterKey,
      salt,
      textEncoder.encode(WRITE_KEY_LABEL),
      KEY_LENGTH,
    ),
  };
}

function additionalData(pathname) {
  return textEncoder.encode(
    `${FORMAT_LABEL}\0${normalizePagePath(pathname)}`,
  );
}

function parseEnvelope(document) {
  if (typeof document !== "string") {
    throw new Error("The private document envelope is malformed.");
  }
  const escapedStart = E2EE_DOCUMENT_START.replace(
    /[.*+?^${}()|[\]\\]/g,
    "\\$&",
  );
  const escapedEnd = E2EE_DOCUMENT_END.replace(
    /[.*+?^${}()|[\]\\]/g,
    "\\$&",
  );
  const match = new RegExp(
    `^${escapedStart}\\r?\\n([^\\r\\n]+)\\r?\\n${escapedEnd}$`,
  ).exec(document);
  if (!match) {
    throw new Error("The private document envelope is malformed.");
  }

  let envelope;
  try {
    envelope = JSON.parse(match[1]);
  } catch {
    throw new Error("The private document envelope is malformed.");
  }
  const fields = Object.keys(envelope ?? {}).sort();
  if (
    fields.join(",") !== "cipher,data,nonce,v" ||
    envelope.v !== 1 ||
    envelope.cipher !== "xchacha20-poly1305"
  ) {
    throw new Error("The private document envelope uses an unsupported format.");
  }

  let nonce;
  let ciphertext;
  try {
    nonce = base64URLToBytes(envelope.nonce, NONCE_LENGTH);
    ciphertext = base64URLToBytes(envelope.data);
  } catch {
    throw new Error("The private document envelope is malformed.");
  }
  if (ciphertext.length < AUTH_TAG_LENGTH) {
    throw new Error("The private document envelope is malformed.");
  }
  return { nonce, ciphertext };
}

export function encryptE2EEDocument(plaintext, contentKey, pathname) {
  if (!(contentKey instanceof Uint8Array) || contentKey.length !== KEY_LENGTH) {
    throw new Error("The private content key is invalid.");
  }
  const nonce = randomBytes(NONCE_LENGTH);
  const ciphertext = xchacha20poly1305(
    contentKey,
    nonce,
    additionalData(pathname),
  ).encrypt(textEncoder.encode(plaintext));
  const payload = JSON.stringify({
    v: 1,
    cipher: "xchacha20-poly1305",
    nonce: bytesToBase64URL(nonce),
    data: bytesToBase64URL(ciphertext),
  });
  return `${E2EE_DOCUMENT_START}\n${payload}\n${E2EE_DOCUMENT_END}`;
}

export function decryptE2EEDocument(document, contentKey, pathname) {
  if (!(contentKey instanceof Uint8Array) || contentKey.length !== KEY_LENGTH) {
    throw new Error("The private content key is invalid.");
  }
  const { nonce, ciphertext } = parseEnvelope(document);
  try {
    return textDecoder.decode(
      xchacha20poly1305(
        contentKey,
        nonce,
        additionalData(pathname),
      ).decrypt(ciphertext),
    );
  } catch {
    throw new Error(
      "This private link has the wrong key, or the ciphertext was changed.",
    );
  }
}

export function privatePageURL(location, encodedMasterKey) {
  base64URLToBytes(encodedMasterKey, KEY_LENGTH);
  const url = new URL(location.href);
  url.search = "";
  url.hash = `key=${encodedMasterKey}`;
  return url.toString();
}

export function createSerialQueue() {
  let tail = Promise.resolve();
  return {
    run(task) {
      const result = tail.then(task);
      tail = result.catch(() => {});
      return result;
    },
    drain() {
      return tail;
    },
  };
}

export function createLatestResultQueue(apply) {
  const serial = createSerialQueue();
  let newest = 0;
  return {
    run(task) {
      const sequence = ++newest;
      return serial.run(task).then((result) => {
        const applied = sequence === newest;
        if (applied) {
          apply(result);
        }
        return { applied, result };
      });
    },
    invalidate() {
      newest++;
    },
    drain() {
      return serial.drain();
    },
  };
}
