CREATE TABLE IF NOT EXISTS trackers
(
    id              TEXT PRIMARY KEY,
    orig_id         TEXT NOT NULL,
    source          TEXT NOT NULL,
    description     TEXT NOT NULL,
    latitude        REAL,
    longitude       REAL
);
