CREATE TABLE IF NOT EXISTS trackers
(
    id              TEXT PRIMARY KEY,
    orig_id         TEXT NOT NULL,
    source          TEXT NOT NULL,
    description     TEXT NOT NULL,
    latitude        REAL,
    longitude       REAL,
    modifiedAt      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER [modifiedAt]
    AFTER UPDATE
    ON trackers
FOR EACH ROW
BEGIN
    UPDATE trackers SET modifiedAt = CURRENT_TIMESTAMP WHERE id = old.id;
END
