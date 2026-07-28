-- name: CreatePage :execrows
INSERT INTO pages (
    title,
    text,
    cursor_start,
    cursor_end,
    published,
    self_destruct,
    locked,
    lock_salt,
    lock_verifier
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9
)
ON CONFLICT (title) DO NOTHING;

-- name: GetPage :one
SELECT title, text, cursor_start, cursor_end, published, self_destruct, locked, lock_salt, lock_verifier
FROM pages
WHERE title = $1
LIMIT 1;

-- name: ConsumeSelfDestructPage :one
DELETE FROM pages
WHERE title = $1 AND self_destruct = TRUE AND locked = FALSE
RETURNING title, text, cursor_start, cursor_end, published, self_destruct, locked, lock_salt, lock_verifier;

-- name: ListPublishedPageTitles :many
SELECT title
FROM pages
WHERE published = TRUE AND self_destruct = FALSE
ORDER BY title;

-- name: UpsertPage :exec
INSERT INTO pages (
    title,
    text,
    cursor_start,
    cursor_end,
    published,
    self_destruct,
    locked,
    lock_salt,
    lock_verifier
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9
)
ON CONFLICT (title) DO UPDATE SET
    text = EXCLUDED.text,
    cursor_start = EXCLUDED.cursor_start,
    cursor_end = EXCLUDED.cursor_end,
    published = EXCLUDED.published,
    self_destruct = EXCLUDED.self_destruct,
    locked = EXCLUDED.locked,
    lock_salt = EXCLUDED.lock_salt,
    lock_verifier = EXCLUDED.lock_verifier,
    updated_at = CURRENT_TIMESTAMP;
