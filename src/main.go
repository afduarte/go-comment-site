package main

import (
	"log"
	"net/http"
	"net/url"
	"time"
	"encoding/json"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/boltdb/bolt"
	"github.com/microcosm-cc/bluemonday"
	"github.com/tidwall/gjson"

	"./db"
	"strings"
	"io/ioutil"
	"errors"
)

// Struct definition

// The message that the server gets and the client sends
// "It's like a jungle sometimes, it makes me wonder how I keep from goin' under"
type Message struct {
	Name      string `json:"name"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Giphy     string `json:"giphy"`
	Upvotes   int    `json:"upvotes"`
}

type LatestComments struct {
	Offset   int        `json:"offset"`
	Messages []Message `json:"messages"`
	IsEnd    bool        `json:"isEnd"`
}

const GIPHY_KEY string = "ec51ed70d5bf4e1385a13a256417a823"

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)

var p = bluemonday.UGCPolicy()

var upgrader = websocket.Upgrader{}

var bucketName = []byte("comments")

// Main function
func main() {
	// Serving files
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	//Web socket
	http.HandleFunc("/ws", handleConnections)

	//Get comments route
	http.HandleFunc("/get-comments", handleGetComments)

	//Upvote
	http.HandleFunc("/upvote", handleUpvote)

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
			if !strings.Contains(err.Error(), "1001") {
				log.Printf("error reading JSON: %v", err)
			}
			delete(clients, ws)
			break
		}
		msg.Name, msg.Message = p.Sanitize(msg.Name), p.Sanitize(msg.Message)
		// Add the timestamp
		msg.Timestamp = getTimestamp()

		msg.Giphy = getGiphy(msg.Giphy)

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
		log.Printf("error: %v", err)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte("{\"error\":\"Error Retrieving latest\"}"))
	} else {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		writer.Write(jsonMsg)
	}
}

// Handle Upvotes
func handleUpvote(writer http.ResponseWriter, request *http.Request) {
	id := request.URL.Query().Get("id")
	if len(id) != 0 {
		jsonMsg, err := upvote(db.BoltDB, id)
		if err == nil {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte(jsonMsg))
		} else {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusNotFound)
			writer.Write([]byte("{error:\"ID: " + id + " NOT FOUND\"}"))
		}
	} else {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte("{error:\"ID PARAMETER MISSING\""))
	}
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

// return the latest 10 messages, taking into account the offset
func getLatest(database *bolt.DB, offset int) ([]byte, error) {
	var jsonMessage []byte
	var finalErr error
	finalErr = database.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket(bucketName)
		stats := b.Stats()
		var response LatestComments
		if stats.KeyN < 1 {
			response = LatestComments{
				Messages: []Message{},
				Offset:   0,
				IsEnd:    true,
			}
			jsonMessage, finalErr = json.Marshal(response)
			if finalErr != nil {
				return finalErr
			}
			return nil
		}
		c := b.Cursor()

		lastKey, _ := c.Last()
		// Prev the Cursor offset amount of times
		lastIndex := 0
		for i := 0; i < offset; i++ {

			tempKey, val := c.Prev()
			if val == nil {
				lastIndex = i + 1
				break
			}
			lastKey = tempKey
		}

		var messages [10]Message

		lastMsgJson := b.Get(lastKey)

		var lastMsg Message

		json.Unmarshal(lastMsgJson, &lastMsg)

		messages[9] = lastMsg
		//Get the latest 5 comments
		end := 0
		for i := 8; i >= 0; i-- {
			_, val := c.Prev()

			if val == nil {
				// the end is current + 1
				end = i + 1
				break
			}

			var msg Message
			json.Unmarshal(val, &msg)
			//put the msg in the end of the array
			messages[i] = msg

		}

		response.Messages = messages[end:]
		if end != 0 {
			response.IsEnd = true
		}
		if lastIndex != 0 {
			response.Offset = lastIndex
		} else {
			response.Offset = offset
		}

		jsonMessage, finalErr = json.Marshal(response)
		if finalErr != nil {
			return finalErr
		}
		return nil
	})
	return jsonMessage, finalErr
}

func upvote(database *bolt.DB, id string) ([]byte, error) {
	var jsonMsg []byte
	err := database.Update(func(tx *bolt.Tx) error {
		key := []byte(id)
		b := tx.Bucket(bucketName)
		val := b.Get(key)
		var err error
		if val == nil {
			return errors.New("UPVOTE: ID not found: " + id)
		}
		var msg Message
		json.Unmarshal(val, &msg)
		msg.Upvotes++
		jsonMsg, err = json.Marshal(msg)
		if err != nil {
			log.Printf("UPVOTE: Error encoding message: %v", err)
			return err
		}
		err = b.Put(key, jsonMsg)
		if err != nil {
			log.Printf("UPVOTE: Error saving message: %v", err)
			return err
		}
		return nil
	})
	return jsonMsg, err
}

func getGiphy(q string) string {
	if len(q) <= 1 {
		return ""
	}
	var gif string
	response, err := http.Get("https://api.giphy.com/v1/gifs/random?rating=R&api_key=" + GIPHY_KEY + "&tag=" + url.QueryEscape(q))
	if err != nil {
		log.Printf("Giphy GET error: %v", err)
	} else {
		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("error getting response body: %v", err)
		}
		//value := gjson.Get(string(body), "data.fixed_width_downsampled_url")
		value := gjson.Get(string(body), "data.image_url")
		gif = value.String()
	}

	return gif
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
