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

// type GameMessage struct {
// 	ID           string  `json:"id"`
// 	Width        float64 `json:"width"`
// 	Height       float64 `json:"height"`
// 	X            float64 `json:"x"`
// 	Y            float64 `json:"y"`
// 	SpeedX       float64 `json:"speedX"`
// 	SpeedY       float64 `json:"speedY"`
// 	Gravity      float64 `json:"gravity"`
// 	GravitySpeed float64 `json:"gravitySpeed"`
// }

type GameMessage struct {
	ID    string  `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Color string  `string:"color"`
}

func main() {
	producer := newProducer()
	defer producer.Close()

	// config websocket server
	http.HandleFunc("/game-input", func(w http.ResponseWriter, r *http.Request) {
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
			var msgo GameMessage
			err = json.Unmarshal(msg, &msgo)
			if err != nil {
				log.Println("Failed to unmarshal message:", err)
			}

			log.Printf("Input server: received msg type : %d from %s sent: %s\n", msgType, conn.RemoteAddr(), string(msg))

			kmsg := &sarama.ProducerMessage{
				Topic: "game-color",
				Key:   sarama.StringEncoder(msgo.ID),
				Value: sarama.ByteEncoder(msg),
			}

			producer.Input() <- kmsg
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../canvas.html")
	})

	log.Println("websocket server starting")
	http.ListenAndServe(":8080", nil)
}

func newProducer() sarama.AsyncProducer {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	config.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	producer, err := sarama.NewAsyncProducer([]string{"188.166.225.204:9092"}, config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	// We will just log to STDOUT if we're not able to produce messages.
	// Note: messages will only be returned here after all retry attempts are exhausted.
	go func() {
		for err := range producer.Errors() {
			log.Println("Failed to write access log entry:", err)
		}
	}()

	return producer
}
