package main

import (
	"log"
	"net/http"
	"time"
	"encoding/json"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/boltdb/bolt"
	"github.com/microcosm-cc/bluemonday"

	"./db"
)

// Struct definition

// The message that the server gets and the client sends
// "It's like a jungle sometimes, it makes me wonder how I keep from goin' under"
type Message struct {
	Name      string `json:"name"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)

var p = bluemonday.UGCPolicy()

var upgrader = websocket.Upgrader{}

var bucketName = []byte("comments")

// Main function
func main() {
	// Serving files
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	//Web socket
	http.HandleFunc("/ws", handleConnections)

	//Get comments route
	http.HandleFunc("/get-comments", handleGetComments)

	// Database (using bolt)
	db.Connect()

	defer db.Close()

	go handleMessages()

	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// Routes

// Handle websocket Connections
func handleConnections(writer http.ResponseWriter, request *http.Request) {
	// Upgrade the request we got to a websocket
	ws, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Fatal("Error Upgrading", err)
	}
	// Close the websocket connection when done
	defer ws.Close()
	clients[ws] = true

	for {
		var msg Message

		// Get new msgs from the ws
		// "Don't push me, 'cause I'm close to the edge."
		err := ws.ReadJSON(&msg)

		// "I'm about to lose my head."
		if err != nil {
			log.Printf("error reading JSON: %v", err)
			delete(clients, ws)
			break
		}
		msg.Name, msg.Message = p.Sanitize(msg.Name), p.Sanitize(msg.Message)
		// Add the timestamp
		msg.Timestamp = getTimestamp()

		broadcast <- msg
	}
}

// Handle GET request for comments
func handleGetComments(writer http.ResponseWriter, request *http.Request) {
	offsetQ := request.URL.Query().Get("offset")
	var offset int
	if len(offsetQ) != 0 {
		var err error
		offset, err = strconv.Atoi(offsetQ)
		if err != nil {
			log.Printf("error parsing offset, ignoring")
			offset = 0
		}
	} else {
		offset = 0
	}
	jsonMsg, err := getLatest(db.BoltDB, offset)
	if err != nil {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte("{\"error\":\"Error Retrieving latest\"}"))
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	writer.Write(jsonMsg)
}

// Helper functions

// Handle messages sent from client
func handleMessages() {
	for {
		// Get the next message from the broadcast channel
		msg := <-broadcast
		// store some data
		go msg.store(db.BoltDB)

		// Send it to all the connected clients
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error writing JSON: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

// Generate Timestamp from current time, in milliseconds
func getTimestamp() int64 {
	return time.Now().Round(time.Millisecond).UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// return the latest 5 messages, taking into account the offset
func getLatest(database *bolt.DB, offset int) ([]byte, error) {
	var jsonMessage []byte
	var finalErr error
	database.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket(bucketName)

		c := b.Cursor()

		lastKey, _ := c.Last()
		// Prev the Cursor offset amount of times
		for i := 0; i < offset; i++ {

			tempKey, val := c.Prev()
			if val == nil {
				break
			}
			lastKey = tempKey
		}

		var messages [5]Message

		lastMsgJson := b.Get(lastKey)

		var lastMsg Message

		json.Unmarshal(lastMsgJson,&lastMsg)

		messages[4] = lastMsg
		//Get the latest 5 comments
		for i := 3; i >= 0; i-- {
			_, val := c.Prev()

			if val == nil {
				log.Printf("end of comments")
				break
			}

			var msg Message
			json.Unmarshal(val, &msg)
			//put the msg in the end of the array
			messages[i] = msg

		}
		jsonMessage, finalErr = json.Marshal(messages)
		if finalErr != nil {
			return finalErr
		}
		return nil
	})

	return jsonMessage, finalErr
}

// Methods

// Persist message on Message struct
func (msg Message) store(db *bolt.DB) error {
	key := []byte(strconv.FormatInt(msg.Timestamp, 10))
	value, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error encoding message: %v", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}

		err = bucket.Put(key, value)
		if err != nil {
			log.Printf("failure writing to DB: %v", err)
			return err
		}

		return nil
	})
	if err != nil {
		log.Printf("Error with transaction: %v", err)
	}
	return err
}
