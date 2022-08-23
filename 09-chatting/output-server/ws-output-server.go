package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

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
	connMap := map[string]*websocket.Conn{}
	// config websocket server
	http.HandleFunc("/chat-output", func(w http.ResponseWriter, r *http.Request) {
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

			// Print the message to the console
			log.Printf("Output server: received msg type : %d from %s sent: %s\n", msgType, conn.RemoteAddr(), string(msg))

			if len(msgo.ID) == 0 {
				continue
			}
			connMap[msgo.ID] = conn
		}
	})

	go http.ListenAndServe(":8091", nil)
	log.Println("websocket server started")

	// config consumer
	keepRunning := true
	log.Println("Starting a new Sarama consumer")
	sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)

	version, err := sarama.ParseKafkaVersion("3.1.0")
	if err != nil {
		log.Panicf("Error parsing Kafka version: %v", err)
	}

	config := sarama.NewConfig()
	config.Version = version
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumer := Consumer{
		connMap: connMap,
		ready:   make(chan bool),
	}

	ctx, cancel := context.WithCancel(context.Background())
	client, err := sarama.NewConsumerGroup([]string{"188.166.225.204:9092"}, "test-chat", config)
	if err != nil {
		log.Panicf("Error creating consumer group client: %v", err)
	}

	consumptionIsPaused := false
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, []string{"chat"}, &consumer); err != nil {
				log.Panicf("Error from consumer: %v", err)
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // Await till the consumer has been set up
	log.Println("Sarama consumer up and running!...")

	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	for keepRunning {
		select {
		case <-ctx.Done():
			log.Println("terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			log.Println("terminating: via signal")
			keepRunning = false
		case <-sigusr1:
			toggleConsumptionFlow(client, &consumptionIsPaused)
		}
	}
	cancel()
	wg.Wait()
	if err = client.Close(); err != nil {
		log.Panicf("Error closing client: %v", err)
	}
}

type Consumer struct {
	connMap map[string]*websocket.Conn
	ready   chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/main/consumer_group.go#L27-L29
	for message := range claim.Messages() {
		log.Printf("Message claimed: key = %s value = %s, timestamp = %v, topic = %s", string(message.Key), string(message.Value), message.Timestamp, message.Topic)

		// message type = 1 means that text message
		msgType := 1

		payload := []byte("Sender: " + string(message.Key) + "| Message: " + string(message.Value))

		// Write message back to browser
		for _, conn := range consumer.connMap {
			if err := conn.WriteMessage(msgType, payload); err != nil {
				continue
			}
		}
		session.MarkMessage(message, "")
	}

	return nil
}

func toggleConsumptionFlow(client sarama.ConsumerGroup, isPaused *bool) {
	if *isPaused {
		client.ResumeAll()
		log.Println("Resuming consumption")
	} else {
		client.PauseAll()
		log.Println("Pausing consumption")
	}

	*isPaused = !*isPaused
}
