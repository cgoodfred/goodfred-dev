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

type sensor struct {
	SID         int     `json:"sensor_id"`
	SensorName  string  `json:"sensor_name"`
	Underweight float32 `json:"underweight_percent"`
	FullWeight  float32 `json:"full_weight"`
}

func (s *sensorReading) getReading(db *sql.DB) error {
	return db.QueryRow("SELECT record_time, weight FROM sensor_reading WHERE sensor_id=$1 ORDER BY record_time DESC",
		s.SID).Scan(&s.Time, &s.Weight)
}

func (s *sensorReading) getLastTenReadings(db *sql.DB) ([]sensorReading, error) {
	sensorReadings := []sensorReading{}
	rows, err := db.Query("SELECT sensor_id, record_time, weight FROM sensor_reading WHERE sensor_id=$1 ORDER BY record_time DESC LIMIT 10", s.SID)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var sens sensorReading
		if err := rows.Scan(&sens.SID, &sens.Time, &sens.Weight); err != nil {
			return sensorReadings, err
		}
		sensorReadings = append(sensorReadings, sens)
	}
	if err = rows.Err(); err != nil {
		return sensorReadings, err
	}
	return sensorReadings, nil
}

func (s *sensorReading) deleteReading(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM sensor_reading WHERE sensor_id=$1", s.SID)

	return err
}

func (s *sensorReading) createReading(db *sql.DB) error {
	_, err := db.Exec("INSERT INTO sensor_reading(sensor_id, record_time, weight) VALUES($1, $2, $3) RETURNING sensor_id", s.SID, s.Time, s.Weight)

	return err
}

func (s *sensor) createSensor(db *sql.DB) error {
	if err := db.QueryRow("INSERT INTO sensor(sensor_name, full_weight, underweight_percent) VALUES($1, $2, $3) RETURNING sensor_id", s.SensorName, s.FullWeight, s.Underweight).Scan(&s.SID); err != nil {
		return err
	}
	return nil
}

func (s *sensor) deleteSensor(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM sensor WHERE sensor_id=$1", s.SID)

	return err
}

func (s *sensor) getSensor(db *sql.DB) error {
	return db.QueryRow("SELECT sensor_id, sensor_name, full_weight, underweight_percent FROM sensor WHERE sensor_id=$1",
		s.SID).Scan(&s.SID, &s.SensorName, &s.FullWeight, &s.Underweight)
}

func (s *sensor) getSensors(db *sql.DB) ([]sensor, error) {
	sensors := []sensor{}
	rows, err := db.Query("SELECT sensor_id, sensor_name, full_weight, underweight_percent FROM sensor")

	defer rows.Close()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var sens sensor
		if err := rows.Scan(&sens.SID, &sens.SensorName, &sens.FullWeight, &sens.Underweight); err != nil {
			return sensors, err
		}
		sensors = append(sensors, sens)
	}
	if err = rows.Err(); err != nil {
		return sensors, err
	}
	return sensors, nil
}
