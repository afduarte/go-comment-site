package db

import (
	"github.com/boltdb/bolt"
	"log"
)

var BoltDB *bolt.DB
var bucketName = []byte("comments")

func Connect() {
	var err error
	BoltDB, err = bolt.Open("./comment-database.db", 0644, nil)
	err = BoltDB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		log.Printf("Connected to DB(%s) : %+v",bucketName,bucket.Stats())
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

func Close() {
	BoltDB.Close()
}