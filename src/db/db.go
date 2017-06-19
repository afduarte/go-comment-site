package db

import (
	"github.com/boltdb/bolt"
	"log"
)

var BoltDB *bolt.DB

func Connect() {
	var err error
	BoltDB, err = bolt.Open("../../database.db", 0644, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func Close() {
	BoltDB.Close()
}