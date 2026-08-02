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
    lock_verifier,
    end_to_end_encrypted,
    e2ee_auth_hash
) VALUES (
    ?1,
    ?2,
    ?3,
    ?4,
    ?5,
    ?6,
    ?7,
    ?8,
    ?9,
    ?10,
    ?11
)
ON CONFLICT (title) DO NOTHING;

-- name: GetPage :one
SELECT title, text, cursor_start, cursor_end, published, self_destruct, locked, lock_salt, lock_verifier, end_to_end_encrypted, e2ee_auth_hash
FROM pages
WHERE title = ?1
LIMIT 1;

-- name: ConsumeSelfDestructPage :one
DELETE FROM pages
WHERE title = ?1 AND self_destruct = TRUE AND locked = FALSE AND end_to_end_encrypted = FALSE
RETURNING title, text, cursor_start, cursor_end, published, self_destruct, locked, lock_salt, lock_verifier, end_to_end_encrypted, e2ee_auth_hash;

-- name: ConsumeE2EESelfDestructPage :one
DELETE FROM pages
WHERE title = ?1 AND self_destruct = TRUE AND locked = FALSE AND end_to_end_encrypted = TRUE
RETURNING title, text, cursor_start, cursor_end, published, self_destruct, locked, lock_salt, lock_verifier, end_to_end_encrypted, e2ee_auth_hash;

-- name: ListPublishedPageTitles :many
SELECT title
FROM pages
WHERE published = TRUE AND self_destruct = FALSE AND end_to_end_encrypted = FALSE
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
    lock_verifier,
    end_to_end_encrypted,
    e2ee_auth_hash
) VALUES (
    ?1,
    ?2,
    ?3,
    ?4,
    ?5,
    ?6,
    ?7,
    ?8,
    ?9,
    ?10,
    ?11
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
    end_to_end_encrypted = excluded.end_to_end_encrypted,
    e2ee_auth_hash = excluded.e2ee_auth_hash,
    updated_at = CURRENT_TIMESTAMP;

-- name: ConvertPageToE2EE :execrows
UPDATE pages
SET text = ?2,
    cursor_start = ?3,
    cursor_end = ?4,
    published = FALSE,
    end_to_end_encrypted = TRUE,
    e2ee_auth_hash = ?5,
    updated_at = CURRENT_TIMESTAMP
WHERE title = ?1
  AND end_to_end_encrypted = FALSE
  AND locked = FALSE
  AND self_destruct = FALSE;
