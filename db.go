package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var db *bolt.DB
var open bool

// Open to create the database and open
func Open(filename string) error {
	var err error
	config := &bolt.Options{Timeout: 30 * time.Second}
	db, err = bolt.Open(filename, 0600, config)
	if err != nil {
		fmt.Println("Opening BoltDB timed out")
		log.Fatal(err)
	}
	open = true
	return nil
}

// Close database
func Close() {
	open = false
	db.Close()
}

// Data for storing in DB
type CowyoData struct {
	Title       string
	CurrentText string
	Diffs       []string
	Timestamps  []string
}

func hasPassword(title string) (bool, error) {
	title = strings.ToLower(title)
	if !open {
		return false, fmt.Errorf("db must be opened before loading!")
	}
	hasPassword := false
	err := db.View(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("datas"))
		if b == nil {
			return fmt.Errorf("db must be opened before loading!")
		}
		k := []byte(title)
		val := b.Get(k)
		if val == nil {
			return nil
		}
		var p CowyoData
		err = p.decode(val)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(p.CurrentText, "\n") {
			if strings.Contains(line, "<") == true && strings.Contains(line, ">") == true && strings.Contains(line, "user") == true && strings.Contains(line, "password") == true && strings.Contains(line, "public") == true {
				hasPassword = true
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Could not get CowyoData: %s", err)
		return false, err
	}
	return hasPassword, nil
}

func (p *CowyoData) load(title string) error {
	title = strings.ToLower(title)
	if !open {
		return fmt.Errorf("db must be opened before loading!")
	}
	err := db.View(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("datas"))
		if b == nil {
			return nil
		}
		k := []byte(title)
		val := b.Get(k)
		if val == nil {
			// make new one
			p.Title = title
			p.CurrentText = ""
			p.Diffs = []string{}
			p.Timestamps = []string{}
			return nil
		}
		err = p.decode(val)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Could not get CowyoData: %s", err)
		return err
	}
	return nil
}

func (p *CowyoData) save(newText string) error {
	if !open {
		return fmt.Errorf("db must be opened before saving")
	}
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("datas"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		// find diffs
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(p.CurrentText, newText, true)
		delta := dmp.DiffToDelta(diffs)
		p.CurrentText = newText
		p.Timestamps = append(p.Timestamps, time.Now().String())
		p.Diffs = append(p.Diffs, delta)
		enc, err := p.encode()
		if err != nil {
			return fmt.Errorf("could not encode CowyoData: %s", err)
		}
		err = bucket.Put([]byte(p.Title), enc)
		return err
	})
	return err
}

func (p *CowyoData) encode() ([]byte, error) {
	enc, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func (p *CowyoData) decode(data []byte) error {
	err := json.Unmarshal(data, &p)
	if err != nil {
		return err
	}
	return nil
}
