ALTER TABLE sensor
DROP COLUMN IF EXISTS is_underweight,
DROP COLUMN IF EXISTS last_reading_weight,
DROP COLUMN IF EXISTS last_reading_time;