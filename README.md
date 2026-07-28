<p align="center">
  <img
    src="web/public/static/logo.jpg"
    width="454"
    alt="cowyo logo: a cow beside a speech bubble saying yo"
  >
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
</p>

*cowyo* is a self-contained shared scratchpad that makes jotting notes easy
and fast. Open a page, type, and it saves automatically. Share the URL to edit
with other people in real time.

Try it at [cowyo.com](https://cowyo.com).

## Getting started

Build and run:

```sh
make build
./cowyo
```

Then open [localhost:8001](http://localhost:8001). Use `-port` to choose a
different port.

cowyo uses SQLite by default. To use PostgreSQL, set `DATABASE_URL` as shown
in `.env.example`. In production, set `SITE_URL` to the site's public origin
(for example, `https://cowyo.com`) so canonical links, social previews,
returned paste URLs, `robots.txt`, and the sitemap always use the authoritative
domain.

### Docker

```sh
docker build -t cowyo .
docker run --name cowyo -p 8001:8001 -v cowyo-data:/data cowyo
```

The repository also includes `disco.json` for deployment with Disco.

## Features

### Editing

Pages save automatically, and people viewing the same page see edits in real
time. Visiting `/` creates a page with a memorable alliterative name such as
`calm-cat`.

### Encryption

Encryption happens entirely in the browser. The password is never sent to the
server, and encrypted text cannot be recovered if the password is lost.
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

curl receives and sends plain text:

```sh
curl https://cowyo.com/my-notes
curl --data-binary @notes.txt https://cowyo.com/
curl --data-binary @notes.txt https://cowyo.com/my-notes
```

## Development

Run the development server:

```sh
make serve
```

Run the tests:

```sh
make test
```

Pull requests are welcome.

## License

MIT
