package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
	Logger *zap.SugaredLogger
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (a *App) Initialize() {
	a.Logger = a.initializeLogger()

	connectionString := getDBConfig()

	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		a.Logger.Fatal(err)
	}

	driver, err := postgres.WithInstance(a.DB, &postgres.Config{})
	m, err := migrate.NewWithDatabaseInstance("github://cgoodfred/goodfred-dev/migrations", "postgres", driver)
	if err != nil {
		a.Logger.Error(err)
	}
	m.Up()

	a.Router = mux.NewRouter()
	a.initializeRoutes()
	a.Router.Use(a.loggingMiddleware)
	a.Logger.Info("initialization successful")
}

func (a *App) Run() {
	originsOK := handlers.AllowedOrigins([]string{"goodfred.dev", "localhost:4785"})
	a.Logger.Fatal(http.ListenAndServe(":8080", handlers.CORS(originsOK)(a.Router)))
}

func getDBConfig() string {
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
	a.Router.HandleFunc("/sensors", a.createSensor).Methods("POST")
	a.Router.HandleFunc("/sensors", a.getSensors).Methods("GET")
	a.Router.HandleFunc("/sensors/underweight", a.getUnderweightSensors).Methods("GET")
	a.Router.HandleFunc("/sensors/{id:[0-9]+}", a.getSensor).Methods("GET")
	// a.Router.HandleFunc("/sensors/{id:[0-9]+}", a.deleteSensor).Methods("DELETE")
	a.Router.HandleFunc("/sensors/{id:[0-9]+}", a.createSensorReading).Methods("POST")
	a.Router.HandleFunc("/sensors/{id:[0-9]+}/readings", a.getSensorReadings).Methods("GET")
	// a.Router.HandleFunc("/sensors/{id:[0-9]+}/readings", a.deleteSensorReading).Methods("DELETE")
}

func (a *App) initializeLogger() *zap.SugaredLogger {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()
	return sugar
}

func (a *App) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		lrw := NewLoggingResponseWriter(w)
		a.Logger.Infow("API request", "Endpoint", r.RequestURI)

		startTime := time.Now()
		next.ServeHTTP(lrw, r)
		endTime := time.Now()
		a.Logger.Infow("API response", "Duration", endTime.Sub(startTime).String(), "StatusCode", lrw.statusCode)
	})
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

func (a *App) getSensorReadings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid sensor ID")
		return
	}

	s := sensor{SID: id}
	readings, err := s.getLastTenReadings(a.DB)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("No records found for sensor_id: %d", id))
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, readings)
}

func (a *App) createSensorReading(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid sensor ID")
		return
	}

	s := sensorReading{SID: id}
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
	if err := s.createReading(a.DB); err != nil {
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
	if err := s.deleteReading(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

func (a *App) createSensor(w http.ResponseWriter, r *http.Request) {

	var s sensor
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&s); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()
	if err := s.createSensor(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, s)
}

func (a *App) deleteSensor(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Sensor ID")
		return
	}

	s := sensor{SID: id}
	if err := s.deleteSensor(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, s)
}

func (a *App) getSensor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid sensor ID")
		return
	}

	s := sensor{SID: id}
	if err := s.getSensor(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("No records found for sensor_id: %d", id))
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	readings, err := s.getLastTenReadings(a.DB)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("No records found for sensor_id: %d", id))
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	sens := sensorResponse{
		Sensor:   s,
		Readings: readings,
	}
	respondWithJSON(w, http.StatusOK, sens)
}

func (a *App) getSensors(w http.ResponseWriter, r *http.Request) {
	s := sensor{}
	sensors, err := s.getSensors(a.DB)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "No sensors found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, sensors)
}

func (a *App) getUnderweightSensors(w http.ResponseWriter, r *http.Request) {
	s := sensor{}
	sensors, err := s.getSensors(a.DB)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "No sensors found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	underweight := []sensor{}
	for _, sens := range sensors {
		if sens.IsUnderweight {
			underweight = append(underweight, sens)
		}
	}

	respondWithJSON(w, http.StatusOK, underweight)
}
