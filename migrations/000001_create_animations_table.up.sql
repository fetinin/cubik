CREATE TABLE IF NOT EXISTS saved_animations (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL,
    name TEXT NOT NULL,
    frames_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_device_id ON saved_animations(device_id);
CREATE INDEX idx_created_at ON saved_animations(created_at DESC);
