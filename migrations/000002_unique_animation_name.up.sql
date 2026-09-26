-- Rename pre-existing duplicates (keep the oldest row's name) so the unique
-- index can be created on databases that predate the constraint.
UPDATE saved_animations
SET name = name || ' (' || substr(id, 1, 8) || ')'
WHERE rowid NOT IN (
    SELECT MIN(rowid) FROM saved_animations GROUP BY device_id, name
);

CREATE UNIQUE INDEX idx_device_name ON saved_animations(device_id, name);
