package main

import (
	"encoding/json"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	opsProcessed = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})
)

func main() {

	db, err := newRecordDB("db/reports.csv")
	if err != nil {
		panic(err)
	}

	peopleDb, err := newPeopleDB("db/people.csv")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		entries := db.records
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = tmpl.Execute(w, entries)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	http.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/new.html"))
		err := tmpl.Execute(w, peopleDb.teams())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	})

	http.HandleFunc("/api/v1/reports", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			out, err := json.Marshal(db.records)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Write(out)
		case "POST":
			entry := statEntry{}
			err := json.NewDecoder(r.Body).Decode(&entry)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			entry.Date = time.Now().UTC().Format("2006/01/02")
			if err = db.store(entry); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				log.Printf("Error storing new entry: %v", err)
				return
			}

			if !peopleDb.hasTeam(entry.Team) {
				if err = peopleDb.store(peopleRecord{Team: entry.Team}); err != nil {
					log.Printf("Error saving team %q: %v", entry.Team, err)
					return
				}
			}
		}
	})
	http.HandleFunc("/api/v1/people", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			out, err := json.Marshal(peopleDb.records)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Write(out)
		}
	})
	http.HandleFunc("/api/v1/teams", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			out, err := json.Marshal(peopleDb.teams())
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Write(out)
		}
	})
	// Support prometheus metrics
	http.Handle("/api/v1/metrics", promhttp.Handler())

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	log.Panic(http.ListenAndServe(":8888", nil))
}
