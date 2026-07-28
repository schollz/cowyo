ALTER TABLE pages
    DROP COLUMN lock_verifier,
    DROP COLUMN lock_salt,
    DROP COLUMN locked;
