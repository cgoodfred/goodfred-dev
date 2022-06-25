package main

import (
	"database/sql"
	"time"
)

type sensorReading struct {
	SID    int       `json:"sensor_id"`
	Time   time.Time `json:"reading_time"`
	Weight float64   `json:"weight"`
}

func (s *sensorReading) getSensorReading(db *sql.DB) error {
	return db.QueryRow("SELECT record_time, weight FROM sensor_reading WHERE sensor_id=$1 ORDER BY record_time DESC",
		s.SID).Scan(&s.Time, &s.Weight)
}

func (s *sensorReading) deleteSensorReading(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM sensor_reading WHERE sensor_id=$1", s.SID)

	return err
}

func (s *sensorReading) createSensorReading(db *sql.DB) error {
	_, err := db.Exec("INSERT INTO sensor_reading(sensor_id, record_time, weight) VALUES($1, $2, $3) RETURNING id", s.SID, s.Time, s.Weight)

	return err
}
