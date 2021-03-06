CREATE TABLE IF NOT EXISTS sensor (
    sensor_id bigserial PRIMARY KEY,
    sensor_name text NOT NULL,
    full_weight real NOT NULL,
    underweight_percent real NOT NULL DEFAULT 50.0,
    UNIQUE(sensor_name)
);

CREATE TABLE IF NOT EXISTS sensor_reading (
    sensor_id bigserial REFERENCES sensor (sensor_id) ON DELETE CASCADE,
    record_time timestamptz default (now() at time zone 'utc'),
    weight real NOT NULL
);