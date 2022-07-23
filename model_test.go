package main

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetSensorReadingSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	testSensorID := 2
	testWeight := 3.0
	testTime := time.Now().UTC().Round(time.Hour)
	expected_sql := "SELECT record_time, weight FROM sensor_reading"

	rows := sqlmock.NewRows([]string{"Time", "Weight"}).AddRow(testTime, testWeight)

	mock.ExpectQuery(expected_sql).WithArgs(testSensorID).WillReturnRows(rows)

	s := sensorReading{
		SID: testSensorID,
	}

	if err = s.getReading(db); err != nil {
		t.Errorf("Unexpected error during delete: %s", err)
	}

	if (s.Weight != testWeight) || (s.Time != testTime) {
		t.Errorf("Values scanned into s for time or weight not correct\n Expected (time,weight): %s, %f \n Actual: %s, %f", testTime.String(), testWeight, s.Time.String(), s.Weight)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled test expecations: %s", err)
	}
}

func TestDeleteSensorReading(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	testSensorID := 1

	mock.ExpectExec("DELETE FROM sensor_reading").WithArgs(testSensorID).WillReturnResult(sqlmock.NewResult(0, 1))

	s := sensorReading{
		SID: testSensorID,
	}

	if err = s.deleteReading(db); err != nil {
		t.Errorf("Unexpected error during delete: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled test expecations: %s", err)
	}
}

func TestCreateSensorReading(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	testSensorID := 5
	testWeight := 6.75
	testTime := time.Now().UTC().Round(time.Hour)
	// Both expected and sensor time are rounded to the hour for simplicity
	mock.ExpectExec("INSERT INTO sensor_reading").WithArgs(testSensorID, testTime, testWeight).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE sensor").WithArgs(testWeight, testTime, testSensorID).WillReturnResult(sqlmock.NewResult(1, 1))

	s := sensorReading{
		SID:    testSensorID,
		Time:   time.Now().UTC().Round(time.Hour),
		Weight: testWeight,
	}

	if err = s.createReading(db); err != nil {
		t.Errorf("Unexpected error during create: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled test expecations: %s", err)
	}
}
