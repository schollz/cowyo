package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
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
	Title string
	Text  string
}

func (p *CowyoData) load() error {
	if !open {
		return fmt.Errorf("db must be opened before saving!")
	}
	err := db.View(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("datas"))
		if b == nil {
			return nil
		}
		k := []byte(p.Title)
		val := b.Get(k)
		if val == nil {
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

func (p *CowyoData) save() error {
	if !open {
		return fmt.Errorf("db must be opened before saving")
	}
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("datas"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
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
