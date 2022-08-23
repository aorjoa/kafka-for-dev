package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"time"

	"github.com/Shopify/sarama"
	"github.com/dghubble/oauth1"
)

type TrendItem struct {
	Name        string `json:"name"`
	Url         string `json:"url"`
	TweetVolume *uint  `json:"tweet_volume"`
}

type TrendList struct {
	Trends []TrendItem `json:"trends"`
}

func main() {
	consumerKey := "someKey"
	consumerSecret := "someSecret"
	accessToken := "someToken"
	accessSecret := "someSecret"

	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)

	httpClient := config.Client(oauth1.NoContext, token)

	// WOEID 23424960 = Thailand
	path := "https://api.twitter.com/1.1/trends/place.json?id=23424960"

	resp, _ := httpClient.Get(path)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var info []TrendList
	json.Unmarshal(body, &info)

	producer := newProducer()
	defer producer.Close()

	for _, trend := range info[0].Trends {
		msh, err := json.Marshal(trend)
		if err != nil {
			log.Fatal(err)
		}
		msg := &sarama.ProducerMessage{
			Topic: "trends",
			Value: sarama.ByteEncoder(msh),
		}
		partition, offset, err := producer.SendMessage(msg)
		log.Printf("partition: %d, offset: %d, err: %s", partition, offset, err)
	}
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
