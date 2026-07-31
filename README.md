<p align="center">
  <a href="https://cowyo.com"><img
    src="web/public/static/logo.jpg"
    width="454"
    alt="cowyo logo: a cow beside a speech bubble saying yo"
  ></a>
</p>

<p align="center">A pastebin for minimalists</p>

<p align="center">
  <a href="https://github.com/schollz/cowyo/actions/workflows/ci.yml">
    <img
      src="https://github.com/schollz/cowyo/actions/workflows/ci.yml/badge.svg?branch=main"
      alt="CI status"
    >
  </a>
  <a href="https://github.com/schollz/cowyo/releases/latest">
    <img
      src="https://img.shields.io/github/v/release/schollz/cowyo"
      alt="Latest release"
    >
  </a>
  <a href="https://github.com/sponsors/schollz"><img alt="GitHub Sponsors" src="https://img.shields.io/github/sponsors/schollz"></a>
</p>

*cowyo* is a self-contained shared scratchpad that makes jotting notes easy
and fast. Open a page, type, and it saves automatically. Share the URL to edit
with other people in real time.

Try it at [cowyo.com](https://cowyo.com).

## Getting started

Build and run:

```sh
make serve
```

Then open [localhost:8001](http://localhost:8001). Use `-port` to choose a
different port.

cowyo uses SQLite by default. To use PostgreSQL, set `DATABASE_URL` as shown
in `.env.example`. In production, set `SITE_URL` to the site's public origin
(for example, `https://cowyo.com`) so canonical links, social previews,
returned paste URLs, `robots.txt`, and the sitemap always use the authoritative
domain. To enable Umami analytics, set `UMAMI_URL` to the Umami origin and
`UMAMI_WEBSITE_ID` to the Cowyo website ID from the Umami dashboard. To load
Google AdSense on browser-rendered pages, set `GOOGLE_ADSENSE` to the
`ca-pub-XXXXXXXXXXXXXXXX` client ID supplied by AdSense.

## Features

### Editing

Pages save automatically, and people viewing the same page see edits in real
time. Each other editor's caret appears as a dim gray line and disappears when
they leave. Visiting `/` creates a page with a memorable alliterative name such
as `calm-cat`.

### Encryption

Encryption happens entirely on the client—the browser does it for the editor.
The password is never sent to the server, and encrypted text cannot be
recovered if the password is lost.
Password fields are hidden by default and have an eye button to reveal them.
On mobile, the password dialog stays inside the visible area as the on-screen
keyboard opens.

### Page locking

A page lock prevents editing without hiding the page's contents. Anyone with
the URL can still read it.

### Publishing

Pages are unpublished by default and excluded from the sitemap. Publishing
makes a page discoverable to search engines and gives it a unique search
description based on its plaintext content. Unpublishing removes the content
from search and social-preview metadata, but does not make its URL private.

### Self-destructing pages

A self-destructing page is returned one final time on its next browser or curl
GET, then deleted.

### Other conveniences

The cow menu can copy the page text and switch between light and dark themes.
Web addresses in the text are clickable.

See [About cowyo](ABOUT.md) for a more complete guide.

## Using curl

cowyo works as a plain-text command-line endpoint without an SDK, JSON
wrapping, or an API token. A GET prints the exact stored text, so it can be
piped to another command or redirected to a file:

```sh
curl https://cowyo.com/my-notes
curl https://cowyo.com/my-notes > notes.txt
```

POST a file to `/` to create a randomly named page. The response is its
shareable URL:

```sh
curl --data-binary @notes.txt https://cowyo.com/
```

POST to a named path to create or replace that page. `@-` reads the body from
stdin:

```sh
curl --data-binary @notes.txt https://cowyo.com/my-notes
printf '%s\n' 'deploy at 3pm' |
  curl --data-binary @- https://cowyo.com/team-handoff
```

A named POST replaces the page's complete contents. Locked pages reject normal
command-line writes until they are unlocked in the browser or through the
page-control API.

### Page-control API

Existing pages can be published, locked, encrypted, or armed for self destruct
through the versioned operations endpoint:

```text
POST /api/v1/pages/{page}/operations
Content-Type: application/json
```

For example, publish a page:

```sh
curl --json '{"operation":"publish"}' \
  https://cowyo.com/api/v1/pages/my-notes/operations
```

The supported operations are `publish`, `unpublish`, `lock`, `unlock`,
`encrypt`, `decrypt`, `self-destruct`, and `cancel-self-destruct`. Lock and
unlock requests include a `password`:

```sh
curl --json '{"operation":"lock","password":"use a strong password"}' \
  https://cowyo.com/api/v1/pages/my-notes/operations
```

Avoid putting a real password in shell history; send JSON from a protected file
or stdin instead. Use HTTPS so page-lock passwords are protected in transit.

Encryption stays client-side. An `encrypt` request supplies one complete,
locally generated `COWYO ENCRYPTED BLOCK V1` in the `text` field; `decrypt`
supplies the locally decrypted result. The API rejects encryption passwords:

```sh
jq -n --rawfile text encrypted.txt \
  '{operation:"encrypt", text:$text}' |
  curl --json @- \
    https://cowyo.com/api/v1/pages/my-notes/operations
```

Successful requests return JSON with the page URL and its current published,
self-destruct, locked, and encrypted states. The endpoint only changes existing
pages, accepts strict JSON, limits transformed text to 16 KiB, and caps the
complete request at 64 KiB.

API mutations share the normal per-client POST limit of 10 per minute with a
burst of 5. Page operations also have per-client and per-page token buckets;
lock and unlock attempts have tighter client-plus-page and page-wide limits to
slow password guessing and distributed state-flipping. Rate-limited responses
return HTTP 429 with `Retry-After`.

## Development

Run the tests:

```sh
make test
```

Pull requests are welcome.

## License

MIT
