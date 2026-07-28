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
    ?1,
    ?2,
    ?3,
    ?4,
    ?5,
    ?6,
    ?7,
    ?8,
    ?9
)
ON CONFLICT (title) DO NOTHING;

-- name: GetPage :one
SELECT title, text, cursor_start, cursor_end, published, self_destruct, locked, lock_salt, lock_verifier
FROM pages
WHERE title = ?1
LIMIT 1;

-- name: ConsumeSelfDestructPage :one
DELETE FROM pages
WHERE title = ?1 AND self_destruct = TRUE AND locked = FALSE
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
    ?1,
    ?2,
    ?3,
    ?4,
    ?5,
    ?6,
    ?7,
    ?8,
    ?9
)
ON CONFLICT (title) DO UPDATE SET
    text = excluded.text,
    cursor_start = excluded.cursor_start,
    cursor_end = excluded.cursor_end,
    published = excluded.published,
    self_destruct = excluded.self_destruct,
    locked = excluded.locked,
    lock_salt = excluded.lock_salt,
    lock_verifier = excluded.lock_verifier,
    updated_at = CURRENT_TIMESTAMP;
