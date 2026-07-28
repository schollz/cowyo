import assert from "node:assert/strict";
import test from "node:test";

import { clickableURL, findURLs } from "../src/links.js";

test("accepts trimmed HTTP and HTTPS URLs", () => {
  assert.equal(
    clickableURL("  https://example.com/path?q=1  "),
    "https://example.com/path?q=1",
  );
  assert.equal(
    clickableURL("\thttp://example.com\t"),
    "http://example.com/",
  );
});

test("rejects non-URL and unsafe URL lines", () => {
  assert.equal(clickableURL("see https://example.com"), null);
  assert.equal(clickableURL("javascript:alert(1)"), null);
  assert.equal(clickableURL("ftp://example.com"), null);
  assert.equal(clickableURL("https://example.com two"), null);
  assert.equal(clickableURL(""), null);
});

test("finds URLs anywhere in the text", () => {
  assert.deepEqual(
    findURLs(
      "Read https://example.com/docs and then http://example.org/a?b=1.",
    ),
    [
      {
        start: 5,
        end: 29,
        text: "https://example.com/docs",
        href: "https://example.com/docs",
      },
      {
        start: 39,
        end: 63,
        text: "http://example.org/a?b=1",
        href: "http://example.org/a?b=1",
      },
    ],
  );
});

test("keeps balanced URL punctuation and excludes prose punctuation", () => {
  const matches = findURLs(
    "(https://example.com/a_(b)), https://example.org/test!",
  );

  assert.deepEqual(
    matches.map((match) => match.text),
    ["https://example.com/a_(b)", "https://example.org/test"],
  );
});

test("ignores unsupported protocols", () => {
  assert.deepEqual(findURLs("javascript:alert(1) ftp://example.com"), []);
});
