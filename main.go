package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type statEntry struct {
	Version      int
	Date         string
	Team         string
	MemberCount  int
	CycleTime    int
	LeadTime     int
	BugsReported int
	BugsSquashed int
	DeployCount  int
	ValueScore   float64
	ReportURL    string

	// Not exported
	QualityScore float64
}

type recordDb struct {
	records  []statEntry
	filename string
}

var (
	opsProcessed = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})
)

func newRecordDB(filename string) (*recordDb, error) {
	csvfile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer csvfile.Close()
	db := &recordDb{
		filename: filename,
	}
	if err = db.load(csvfile); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *recordDb) load(f io.Reader) error {
	entries := []statEntry{}

	r := csv.NewReader(f)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(record) != 11 {
			continue
		}

		entry := statEntry{
			Date:      record[1],
			Team:      record[2],
			ReportURL: record[10],
		}

		if entry.MemberCount, err = strconv.Atoi(record[3]); err != nil {
			log.Printf("Unable to convert member count %q to number: %v", record[3], err)
		}
		if entry.CycleTime, err = strconv.Atoi(record[4]); err != nil {
			log.Printf("Unable to convert cycle time %q to number: %v", record[4], err)
		}
		if entry.LeadTime, err = strconv.Atoi(record[5]); err != nil {
			log.Printf("Unable to convert lead time %q to number: %v", record[5], err)
		}
		if entry.BugsReported, err = strconv.Atoi(record[6]); err != nil {
			log.Printf("Unable to convert bugs reported %q to number: %v", record[6], err)
		}
		if entry.BugsSquashed, err = strconv.Atoi(record[7]); err != nil {
			log.Printf("Unable to convert bugs squashed %q to number: %v", record[7], err)
		}
		if entry.DeployCount, err = strconv.Atoi(record[8]); err != nil {
			log.Printf("Unable to convert deploy count %q to number: %v", record[8], err)
		}
		if entry.ValueScore, err = strconv.ParseFloat(record[9], 64); err != nil {
			log.Printf("Unable to convert value score %q to number: %v", record[9], err)
		}
		if entry.BugsSquashed > 0 {
			entry.QualityScore = float64(entry.BugsReported) / float64(entry.BugsSquashed)
		} else {
			entry.QualityScore = float64(-entry.BugsReported)
		}

		entries = append(entries, entry)
	}
	db.records = entries
	return nil
}

func (db *recordDb) validateRecord(r statEntry) error {
	if r.Team == "" {
		return errors.New("missing team name")
	}
	if len(r.Team) > 10 {
		return errors.New("team name must be at most 10 characters long")
	}
	if r.MemberCount < 1 {
		return errors.New("team must have at least one participating member")
	}
	if r.CycleTime < 1 {
		return errors.New("missing cycle time")
	}
	if r.LeadTime < 1 {
		return errors.New("missing lead time")
	}
	if r.BugsReported < 0 {
		return errors.New("bugs reported can't be under 0")
	}
	if r.BugsSquashed < 0 {
		return errors.New("squashed bugs can't be under 0")
	}
	return nil
}

func (db *recordDb) store(entry statEntry) error {
	if err := db.validateRecord(entry); err != nil {
		return err
	}
	if entry.BugsSquashed > 0 {
		entry.QualityScore = float64(entry.BugsReported) / float64(entry.BugsSquashed)
	} else {
		entry.QualityScore = float64(-entry.BugsReported)
	}
	db.records = append(db.records, entry)
	csvfile, err := os.Create(db.filename)
	if err != nil {
		return err
	}
	defer csvfile.Close()

	for i := range db.records {
		record := fmt.Sprintf("%s,%s,%d,%d,%d,%d,%d,%d,%.2f,%s\n", db.records[i].Date, db.records[i].Team, db.records[i].MemberCount, db.records[i].CycleTime, db.records[i].LeadTime, db.records[i].BugsReported, db.records[i].BugsSquashed, db.records[i].DeployCount, db.records[i].ValueScore, db.records[i].ReportURL)
		csvfile.Write([]byte(record))
	}
	csvfile.Sync()
	return nil
}

func main() {

	db, err := newRecordDB("db/reports.csv")
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
		err := tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	})

	http.HandleFunc("/api/v1/entry", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
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
			}

		}
	})
	http.HandleFunc("/api/v1/export", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			out, err := json.Marshal(db.records)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Write(out)
		}
	})
	// Support prometheus metrics
	http.Handle("/api/v1/metrics", promhttp.Handler())
	log.Panic(http.ListenAndServe(":8888", nil))
}
