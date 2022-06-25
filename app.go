package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
}

func (a *App) Initialize() {
	connectionString := db_config()

	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	a.Router = mux.NewRouter()

	a.initializeRoutes()
}

func (a *App) Run() {
	log.Fatal(http.ListenAndServe(":8080", a.Router))
}

func db_config() string {
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "postgres")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "15432")
	viper.SetDefault("DB_NAME", "sensor")
	host := viper.Get("DB_HOST")
	port := viper.GetInt("DB_PORT")
	user := viper.Get("DB_USER")
	password := viper.Get("DB_PASSWORD")
	dbname := viper.Get("DB_NAME")

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/sensor", a.createSensorReading).Methods("POST")
	a.Router.HandleFunc("/sensor/{id:[0-9]+}", a.getSensorReading).Methods("GET")
	a.Router.HandleFunc("/sensor/{id:[0-9]+}", a.deleteSensorReading).Methods("DELETE")
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (a *App) getSensorReading(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid sensor ID")
		return
	}

	s := sensorReading{SID: id}
	if err := s.getSensorReading(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("No records found for sensor_id: %d", id))
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, s)
}

func (a *App) createSensorReading(w http.ResponseWriter, r *http.Request) {

	var s sensorReading
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&s); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	// if a payload has no time, default to current UTC time
	if s.Time.IsZero() {
		s.Time = time.Now().UTC()
	}
	defer r.Body.Close()
	if err := s.createSensorReading(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, s)
}

func (a *App) deleteSensorReading(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Sensor ID")
		return
	}

	s := sensorReading{SID: id}
	if err := s.deleteSensorReading(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}
