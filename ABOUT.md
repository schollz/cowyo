# About cowyo

cowyo is a minimalist shared scratchpad. Every page lives at a simple URL: open one, type, and your text is saved automatically. When other people are editing the same page, their carets appear as dim gray lines.

The cow menu provides the page's controls:

- Copy the complete page text to the clipboard.
- Lock or unlock editing.
- Encrypt or decrypt text with a password.
- Irreversibly convert an ordinary page to permanent end-to-end encryption.
- Publish or unpublish the page.
- Switch between light and dark themes.
- Make the page self-destruct after its next load.

The landing page also has a **Private scratchpad** action for pages that are
end-to-end encrypted from their first save.

## Visibility, locks, and two kinds of encryption

These features serve different purposes:

Unpublished is the default. It keeps a page out of cowyo's sitemap and asks search engines not to index it, but it does not prevent someone with the URL from opening it. Publishing makes the page discoverable to search engines; unpublishing does not change its text.

Page locking prevents changes until the page-lock password is entered. It does not hide or encrypt the text. A locked page remains readable, and its password cannot be recovered if forgotten. The password is sent to cowyo when you lock or unlock the page, but is never stored as plain text.

Password-encrypted blocks protect selected text on an ordinary page. Encryption
and decryption happen in the browser, and the password is never sent to the
server. The server stores a `COWYO ENCRYPTED BLOCK V1`; ordinary text before or
after a block remains ordinary. If you lose the encryption password, the
encrypted text cannot be recovered.

Permanent private mode encrypts the complete document before every save. The
browser generates a random 256-bit master key in a `#key` URL fragment and
derives independent content-encryption and write-capability keys. The server
receives ciphertext and the write capability, stores only a hash of that
capability, and never receives the fragment or derived content key. The full
private URL grants read and write access and cannot be recovered if lost.

Private-from-creation pages never send plaintext to cowyo. Conversion is
irreversible but not retroactive: plaintext previously sent for an ordinary
page may remain in database history, backups, or logs. Private pages cannot be
published, changed by curl or the page-control API, or use password-encrypted
blocks. A server capable of replacing the JavaScript it delivers is outside
this browser-client threat model.

Unlock a page before encrypting it or changing whether it is published. This
can be done in the browser or through the page-control API.

## Self-destructing pages

The bomb action arms a page to self-destruct. For an ordinary page, the next
browser or curl GET receives the page one final time and deletes it; later
visits open an empty page. For a private page, keyless GETs reveal and consume
nothing. After write-capability authentication, exactly one authorized browser
receives the final ciphertext over WebSocket and atomically deletes the page.
A HEAD request does not consume either kind.

Arming self-destruct automatically unpublishes the page. It cannot be combined with publishing, and you must unlock a locked page before arming or cancelling it.

## Using cowyo with curl

A normal curl GET returns only the stored text, without the browser
editor:

```sh
curl https://cowyo.com/my-notes
```

A private page returns its ciphertext envelope. Because the fragment key is
never sent in HTTP requests, curl cannot decrypt or mutate a private page.

POST to the home page to create a new randomly named page:

```sh
curl --data-binary @notes.txt https://cowyo.com/
```

POST to a named path to create that page or replace its complete
contents:

```sh
curl --data-binary @notes.txt https://cowyo.com/my-notes
```

For a short note, the text can be supplied directly:

```sh
curl --data-binary 'remember the milk' https://cowyo.com/shopping-list
```

`--data-binary` preserves the text exactly, including line breaks.

## Controlling a page with the API

The versioned page-control endpoint works with existing pages:

```text
POST /api/v1/pages/{page}/operations
Content-Type: application/json
```

Publish or arm a page for self destruct by sending an operation:

```sh
curl --json '{"operation":"publish"}' \
  https://cowyo.com/api/v1/pages/my-notes/operations

curl --json '{"operation":"self-destruct"}' \
  https://cowyo.com/api/v1/pages/my-notes/operations
```

The complete operation list is `publish`, `unpublish`, `lock`, `unlock`,
`encrypt`, `decrypt`, `self-destruct`, and `cancel-self-destruct`. Lock and
unlock requests also supply `password`. Send password-bearing JSON from a
protected file or stdin rather than leaving the password in shell history, and
use HTTPS.

Encryption remains local: create or decrypt the signed
`COWYO ENCRYPTED BLOCK V1` on the client, then send the transformed content in
the API's `text` field. The encryption password is rejected if sent to cowyo.

The API accepts strict JSON, operates only on existing pages, limits transformed
text to 16 KiB, and returns HTTP 429 with `Retry-After` when a client, page, or
page-lock attempt exceeds its rate limit.

Permanent private pages reject every page-control API mutation.
