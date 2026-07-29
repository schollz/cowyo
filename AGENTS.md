# AGENTS.md

This file is the working guide for contributors and coding agents. It describes
how cowyo2 currently behaves, where its source of truth lives, and how to
validate changes.

> Maintenance rule: whenever repository behavior, architecture, build flow,
> tooling, configuration, generated code, or conventions change, update this
> `AGENTS.md` in the same change so it stays accurate.

## Git workflow

When the repository owner asks to commit and push, commit directly to `main`
and push `main` to `origin` without creating a pull request. Use a branch or
pull request only when the owner explicitly asks for one.

## Repository overview

cowyo2 is a minimalist pastebin with a Go HTTP/WebSocket server and a
Vite-built browser client. The optimized frontend is generated into
`internal/site/build/` and embedded into the Go binary with `//go:embed`.

Important paths:

- `cmd/cowyo/`: production server command entrypoint.
- `internal/cowyo/server.go`: server startup, routing, browser rendering, curl
  plaintext responses, and WebSocket editing.
- `Dockerfile`: multi-stage Disco image build for the Vite frontend and Go
  server.
- `disco.json`: Disco web-service, health-check, and persistent SQLite-volume
  configuration.
- `.github/workflows/ci.yml`: GitHub Actions test workflow for pushes to
  `main` and pull requests.
- `.github/workflows/lock-closed-threads.yml`: daily and manually triggered
  workflow that comments on and locks closed issues.
- `.github/ISSUE_TEMPLATE/`: required bug-report issue form and issue chooser
  configuration; blank issues are disabled.
- `cmd/migrate/`: migration-only command used by `make migrate`.
- `internal/cowyo/post.go`: curl POST handling, the 16 KiB body limit,
  per-client rate limiting, random-page allocation, and returned URLs.
- `internal/cowyo/page_lock.go`: scrypt page-lock verifier creation and
  checking.
- `internal/cowyo/names.go`: adjective and animal lists plus the alliterative
  random-name generator.
- `internal/site/assets.go`: optimized frontend embedding and filesystem access.
- `web/index.html`: the Go-compatible HTML template for the dedicated landing
  and About pages plus the editor UI.
- `web/src/main.js`: shared Vite entrypoint that loads the site styles and
  initializes the editor only on paste pages.
- `web/src/editor.js`: editor synchronization, save/cow menu, password modal,
  theme selection, encryption actions, and page lock/unlock interactions.
- `web/src/theme.js`: system theme detection and locally cached light/dark
  overrides.
- `web/src/encryption.js`: versioned client-side encrypted-block format.
- `web/src/links.js`: URL detection and link-overlay rendering.
- `web/src/style.css`: editor, menu, and modal styling.
- `web/public/static/`: static icons, manifest, font, the vendored cowyo logo, and
  its derived social-preview image copied into the Vite output.
- `web/tests/`: Node tests for browser-side modules.
- `internal/database/`: storage abstraction, migrations, queries, and
  sqlc-generated packages.

## Build and development

The supported production build is:

```sh
make build
```

`make build`:

1. Runs `npm ci` in `web/` when `web/node_modules/.package-lock.json` is absent
   or stale.
2. Runs the optimized Vite build into `internal/site/build/`.
3. Builds `cmd/cowyo` as the `cowyo` binary with the generated frontend
   embedded.

The Go embed requires `internal/site/build/` to exist, so use the Make targets
on a clean checkout instead of invoking `go build` first.

Other commands:

```sh
make test       # build frontend, run Node tests, then Go tests
make frontend   # build only the optimized Vite frontend
make generate   # regenerate both sqlc query packages
make migrate    # apply pending database migrations without starting the server
make serve      # watch frontend/Go files, rebuild, and restart the server
npm --prefix web run dev  # Vite development server
```

`make serve` runs the pinned Air version declared in `Makefile`. Air performs a
complete `make build` initially and whenever Go files, Vite frontend sources,
or public website assets change, then runs or restarts `cowyo` on its default
port (`8001`) with `-log debug`. Generated `internal/site/build/`,
`web/node_modules/`, and `tmp/` content is excluded from watching to prevent
rebuild loops.

The sqlc version is pinned in `Makefile`. Go and npm dependency versions are
declared in `go.mod`, `web/package.json`, and `web/package-lock.json`.

## Runtime configuration and storage

The server loads `.env` at startup.

- Set `DATABASE_URL` to use PostgreSQL.
- If `DATABASE_URL` is empty, SQLite is used.
- `SQLITE_PATH` defaults to `cowyo2.sqlite3`.
- `SITE_URL` optionally sets the authoritative public HTTP(S) origin used for
  canonical URLs, social images, returned paste URLs, `robots.txt`, and the
  sitemap. It must not contain a path, query string, fragment, or credentials.
- A non-empty `ADMIN_POST_KEY` authorizes unrestricted HTTP POSTs when the
  request supplies the exact value in `X-Cowyo-Admin-Key`. Keep it secret and
  deploy behind HTTPS.
- `-log` selects the logging level and defaults to `info`.

Database migrations run automatically when the store opens. They can also be
applied without starting the server by running `make migrate`. PostgreSQL and
SQLite have separate migrations and query files:

```text
internal/database/migrations/postgresql/
internal/database/migrations/sqlite/
internal/database/queries/postgresql/
internal/database/queries/sqlite/
```

When changing the schema or query behavior:

1. Update both database dialects.
2. Add paired up/down migrations when the schema changes.
3. Run `make generate`.
4. Do not hand-edit files under `internal/database/postgresdb/` or
   `internal/database/sqlitedb/`; sqlc owns them.
5. Test with `CGO_ENABLED=0` because the SQLite driver is pure Go.

## HTTP and editing behavior

- A browser `GET /` renders the indexable marketing landing page.
- The landing page's start action, `GET /?new=1`, redirects to a random
  alliterative `adjective-animal` path such as `/calm-cat`, using common
  adjectives and animal names. A curl `GET /` retains the same redirect for
  command-line compatibility.
- `GET /about` renders the indexable About page for browser and command-line
  user agents. `/about` is a reserved site route and rejects POSTs.
- `GET /name` returns the browser editor normally.
- A `curl/*` user agent requesting `/name` receives only the paste's plaintext
  body with a `text/plain` content type.
- An unlocked page armed for self destruct is returned by exactly one final
  browser or curl GET and atomically deleted as part of that load. HEAD does
  not consume it, and the final response uses `Cache-Control: no-store`.
- `GET /sitemap.xml` lists the home and About pages plus pages whose persisted
  publication flag is set, using absolute URLs from `SITE_URL` or the request
  host and forwarded protocol.
- `GET /robots.txt` allows crawling and advertises `/sitemap.xml`.
- Published page responses send `index, follow`; unpublished and missing page
  responses send `noindex, nofollow`.
- Every browser page includes an absolute canonical URL, description, Open
  Graph and X card metadata, and a Schema.org JSON-LD graph describing cowyo
  and the current page. Published pastes use a bounded plaintext excerpt and
  `article` Open Graph type; unpublished pages use generic site copy so their
  contents do not leak into link-preview metadata.
- Social previews and structured data use the official vendored cowyo logo at
  `/static/logo.jpg`; `/static/og.jpg` is the centered 1200×630 derivative for
  large link cards.
- Canonical URLs use `SITE_URL` when configured and otherwise fall back to the
  request host and forwarded protocol.
- `POST /` creates a new alliterative random document atomically and returns
  its absolute URL in the response body and `Location` header.
- `POST /name` creates or replaces the named document and returns its URL.
- `POST /name` returns HTTP 423 and does not modify an existing locked page.
- POST bodies are limited to 16 KiB.
- POST requests are rate limited per client IP to 10 per minute with a burst of
  5.
- POSTs with a configured, matching `X-Cowyo-Admin-Key` bypass the rate limit,
  body limit, and locked-page write restriction. Admin replacements preserve
  the page's publication, self-destruct, and page-lock metadata.
- `/ws?place=name` carries browser edits and broadcasts updates to other
  editors on the same document.
- WebSocket page mutations are serialized with named POST lock checks so a
  write cannot race a page-lock change within one server process.
- In debug mode, every successful POST, WebSocket mutation, or self-destruct
  consumption logs the page path, mutation source, operation, and byte count
  without logging page contents or passwords.
- Non-root paths with a trailing slash are permanently redirected to the
  slashless path.

Random POST creation uses an atomic database insert and retries name
collisions. Keep browser and POST random naming on the shared generator in
`internal/cowyo/names.go`.

## Frontend behavior and encryption

The dedicated home page explains cowyo's zero-account shared-scratchpad
workflow, links to the About page and source repository, previews the editor,
and highlights memorable URLs, privacy controls, and curl access. Its primary
action starts a new randomly named scratchpad. The footer links to schollz's
GitHub Sponsors page, About, and the source repository, and offers an
expandable list of other tools (`croc`, `wthrtxt`, and `yesnotice`). The
dedicated About page uses the same site shell and explains the three-step
workflow, the distinct unpublished, locked, encrypted, and self-destruct
states, curl usage, and the open-source project. Both pages follow the same
system or locally selected
light/dark theme as the editor.

The editor autosaves through the WebSocket. The cow action icon is labeled
`yo`, remains visible at reduced opacity, becomes fully dark on editor input,
stays dark as typing continues, and returns to its resting appearance one
second after the latest input. This visual timer is independent of the 100 ms
save debounce. The menu stays open until the user clicks away. It lists the
current-page actions in this order: copy-text, encrypt/decrypt,
publish/unpublish, page lock/unlock, and self-destruct, followed directly by
the device-wide theme action and About link to `/about`. Opening the menu
focuses the copy action. The site
initially follows the system light/dark preference; choosing the sun/moon action
stores a site-wide browser override in local storage. Icon controls use fast
custom speech-bubble tooltips styled like the menu rather than browser-native
title tooltips. Each tooltip's outlined side opens into an unfilled triangular
point toward its action. Copying briefly replaces the copy icon with a
high-contrast checkmark and announces success through the live status region.

Encryption happens entirely in the browser:

- Passwords are not persisted or sent to the server.
- Encrypting requires two matching password entries; decrypting requires one.
- Each password field is hidden by default and has its own Lucide closed/open
  eye control for revealing or hiding its value.
- The password input is mounted only while its dialog is open and uses
  `data-1p-ignore`/`data-op-ignore`, so 1Password does not mistake encryption
  or page-lock secrets for website login credentials.
- Password dialogs follow the mobile visual viewport and cap their height so
  password fields remain reachable when helper UI and the keyboard are open.
- scrypt derives a key from the password and a random salt.
- XChaCha20-Poly1305 provides authenticated encryption with a random nonce.
- Ciphertext is wrapped in a versioned
  `COWYO ENCRYPTED BLOCK V1` begin/end signature.
- Decryption replaces only complete signed blocks, preserving any ordinary
  text around them.

The server never receives encryption passwords or decrypts content. It stores
encrypted blocks as paste text; while a page is locked, it recognizes the
versioned encrypted-block markers only to restrict the decryption exception
described below.

Page locking is separate from encryption:

- A page is unlocked by default.
- Locking requires two matching password entries; unlocking requires one.
- Locking stores a boolean lock state, random salt, and scrypt password
  verifier in the page's database record. Lock metadata is not part of the
  paste text. This is edit protection, not encryption.
- The page-lock password is sent to the server for a lock or unlock request,
  checked server-side, and never stored as plaintext. Deploy behind HTTPS/WSS
  so credentials are protected in transit.
- The server-provided lock state makes the textarea read-only. Ordinary
  WebSocket edits and named curl POSTs are rejected until the correct password
  clears the database lock metadata.
- Curl GET is normally non-mutating and returns only the exact stored paste
  text. The one exception is the final load of an unlocked self-destruct page.
- Encryption is rejected while a page is locked; unlock it first.
- Client-side decryption remains available on an already-encrypted locked
  page. The server preserves the database lock metadata while requiring
  decryption to preserve all text outside existing complete encrypted blocks.
- A lock verifier cannot recover a forgotten password. Like any stored
  password verifier, it permits offline password guessing, so encourage strong
  passwords.

Page publishing is separate from content and page locking:

- Pages are unpublished by default and excluded from `/sitemap.xml`.
- The globe action publishes or unpublishes through the WebSocket operation
  channel, and the server broadcasts the resulting state to other editors.
- Ordinary WebSocket edits cannot forge publication state. Named curl POSTs
  preserve an existing page's publication state.
- Publication changes are rejected while the page is locked; unlock it first.
- Arming self destruct unpublishes the page, and publishing remains blocked
  until self destruct is cancelled.
- `/robots.txt` advertises the dynamically generated sitemap. The home and
  About pages are listed first, followed by published paste entries ordered by
  page title for deterministic output.

Page self destruct is persisted separately from content:

- The bomb action arms or cancels self destruct through the WebSocket operation
  channel, and ordinary edits or named curl POSTs preserve the stored state.
- Arming and cancelling are rejected while the page is locked. Locking an
  already armed page cancels self destruct.
- The next unlocked GET uses an atomic database delete-and-return operation, so
  concurrent requests cannot both receive the final load. Both browser and curl
  GETs consume it; HEAD does not.
- Armed pages are unpublished, excluded from `/sitemap.xml`, marked
  `noindex, nofollow`, and served with `Cache-Control: no-store` on the final
  load.

## Tests and validation

Tests are organized as follows:

- `internal/cowyo/*_test.go`: server, curl, WebSocket, naming, page locking,
  limits, and build integration tests.
- `internal/database/*_test.go`: backend-neutral storage tests.
- `web/tests/*.test.js`: frontend URL, encryption, theme, and save-activity
  tests.

For normal changes, run:

```sh
make test
make build
```

The CI GitHub Actions workflow runs `make test` on Ubuntu for every pull
request and push to `main`, using the Go version declared in `go.mod` and
Node.js 24. A separate daily workflow comments on and locks closed issues with
no inactivity delay and can also be run manually. The README shows the CI
workflow status for `main` and the repository's latest GitHub release.

For changes touching concurrency, storage, dependencies, security, or build
behavior, also run:

```sh
npm --prefix web audit
go mod verify
go vet ./...
go test -race ./...
CGO_ENABLED=0 go test ./...
git diff --check
```

Add or update tests whenever behavior changes. Security-sensitive frontend
changes should cover successful round trips, wrong credentials, malformed
input, and tampering as applicable.

## Generated and local-only files

Do not commit local build/runtime artifacts:

- `.env` and local `.env.*` variants (except `.env.example`)
- `internal/site/build/`
- `cowyo`
- `web/node_modules/`
- SQLite database (`*.db`, `*.sqlite`, or `*.sqlite3`), WAL, or shared-memory
  files
- npm debug logs
- macOS `.DS_Store` metadata
- `tmp/`

`internal/site/build/` is deliberately ignored even though it is embedded into
the binary; the Makefile recreates it before compiling Go.

## Disco deployment

Disco deployments build the root `Dockerfile`. Its Node stage runs the Vite
production build, its Go stage compiles a static Linux binary with the
generated frontend embedded, and its Alpine runtime runs that binary as the
unprivileged `cowyo` user on port `8001`.

The root `disco.json` exposes the reserved `web` service on port `8001`, checks
readiness through `/robots.txt`, and mounts the named `cowyo2-data` volume at
`/data`. The image defaults `SQLITE_PATH` to `/data/cowyo2.sqlite3`, so the
default SQLite database is persisted across container replacements. A custom
SQLite path must remain under `/data` to be persistent.

For PostgreSQL deployments, set `DATABASE_URL` through the Disco dashboard or
CLI. Set `ADMIN_POST_KEY` there as well. Never put secrets in `disco.json`;
Disco commits that file with the repository. The application applies database
migrations during startup, so the Disco configuration does not define a
separate migration hook.

## Change checklist

Before handing off a change:

1. Keep PostgreSQL and SQLite behavior aligned.
2. Regenerate sqlc output after query or schema changes.
3. Keep curl and browser behavior compatible where they share a route.
4. Preserve the 16 KiB POST limit and posting rate limiter unless the task
   intentionally changes them.
5. Keep generated frontend output and binaries out of version control.
6. Update `README.md` when user-facing setup or behavior changes.
7. Update this `AGENTS.md` whenever any documented fact or workflow changes.
8. Run validation appropriate to the risk of the change.
