package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

type peopleRecord struct {
	Version int
	Name    string
	Team    string
}

type peopleDb struct {
	filename string
	records  []peopleRecord
}

func (db *peopleDb) teams() []string {
	tmpMap := make(map[string]bool)
	teams := []string{}

	for i := range db.records {
		tmpMap[db.records[i].Team] = true
	}

	for k, _ := range tmpMap {
		teams = append(teams, k)
	}

	return teams
}

func newPeopleDB(filename string) (*peopleDb, error) {
	csvfile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer csvfile.Close()
	db := &peopleDb{
		filename: filename,
	}
	if err = db.load(csvfile); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *peopleDb) hasTeam(team string) bool {
	teams := db.teams()
	for i := range teams {
		if teams[i] == team {
			return true
		}
	}
	return false
}

func (db *peopleDb) load(f io.Reader) error {
	entries := []peopleRecord{}

	r := csv.NewReader(f)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(record) != 3 {
			continue
		}

		entry := peopleRecord{
			Name: record[1],
			Team: record[2],
		}
		entries = append(entries, entry)
	}
	db.records = entries
	return nil
}

func (db *peopleDb) store(entry peopleRecord) error {
	db.records = append(db.records, entry)
	csvfile, err := os.Create(db.filename)
	if err != nil {
		return err
	}
	defer csvfile.Close()

	for i := range db.records {
		record := fmt.Sprintf("%d,%s,%s\n", db.records[i].Version, db.records[i].Name, db.records[i].Team)
		csvfile.Write([]byte(record))
	}
	csvfile.Sync()
	return nil
}
