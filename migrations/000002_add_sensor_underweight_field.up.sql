ALTER TABLE sensor
ADD COLUMN is_underweight boolean DEFAULT false,
ADD COLUMN last_reading_weight real NOT NULL DEFAULT 0.0,
ADD COLUMN last_reading_time timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00 Z';