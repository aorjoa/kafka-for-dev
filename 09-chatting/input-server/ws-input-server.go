package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Shopify/sarama"
	"github.com/gorilla/websocket"
)

type TrendItem struct {
	Name        string `json:"name"`
	Url         string `json:"url"`
	TweetVolume *uint  `json:"tweet_volume"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type ChatMessage struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func main() {
	producer := newProducer()
	defer producer.Close()

	// config websocket server
	http.HandleFunc("/chat-input", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Fatal(err)
		}
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msgo ChatMessage
			err = json.Unmarshal(msg, &msgo)
			if err != nil {
				log.Println("Failed to unmarshal message:", err)
			}

			log.Printf("Input server: received msg type : %d from %s sent: %s\n", msgType, conn.RemoteAddr(), string(msg))

			kmsg := &sarama.ProducerMessage{
				Topic: "chat",
				Key:   sarama.StringEncoder(msgo.ID),
				Value: sarama.ByteEncoder(msgo.Message),
			}

			partition, offset, err := producer.SendMessage(kmsg)
			log.Printf("partition: %d, offset: %d, err: %s", partition, offset, err)

		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../chat-panel.html")
	})

	log.Println("websocket server starting")
	http.ListenAndServe(":8090", nil)
}

func newProducer() sarama.SyncProducer {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal // Only wait for the leader to ack
	config.Producer.Return.Successes = true
	config.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	producer, err := sarama.NewSyncProducer([]string{"188.166.225.204:9092"}, config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	return producer
}
