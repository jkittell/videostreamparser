package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/wagslane/go-rabbitmq"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

var url = "http://devimages.apple.com/iphone/samples/bipbop/bipbopall.m3u8"

func TestSegments(t *testing.T) {
	conn, err := rabbitmq.NewConn(
		os.Getenv("RABBITMQ_URL"),
		rabbitmq.WithConnectionOptionsLogging,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	publisher, err := rabbitmq.NewPublisher(
		conn,
		rabbitmq.WithPublisherOptionsLogging,
		rabbitmq.WithPublisherOptionsExchangeName(os.Getenv("RABBITMQ_EXCHANGE_NAME")),
		rabbitmq.WithPublisherOptionsExchangeDeclare,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer publisher.Close()

	publisher.NotifyReturn(func(r rabbitmq.Return) {
		log.Printf("message returned from server: %s", string(r.Body))
	})

	publisher.NotifyPublish(func(c rabbitmq.Confirmation) {
		log.Printf("message confirmed from server. tag: %v, ack: %v", c.DeliveryTag, c.Ack)
	})

	// block main thread - wait for shutdown signal
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	fmt.Println("awaiting signal")

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			job := Job{
				Id:  uuid.New(),
				URL: url,
			}
			data, err := json.Marshal(job)
			if err != nil {
				t.Fatal(err)
			}
			err = publisher.PublishWithContext(
				context.Background(),
				data,
				[]string{os.Getenv("RABBITMQ_ROUTING_KEY")},
				rabbitmq.WithPublishOptionsContentType("application/json"),
				rabbitmq.WithPublishOptionsMandatory,
				rabbitmq.WithPublishOptionsPersistentDelivery,
				rabbitmq.WithPublishOptionsExchange(os.Getenv("RABBITMQ_EXCHANGE_NAME")),
			)
			if err != nil {
				log.Println(err)
			}
		case <-done:
			fmt.Println("stopping publisher")
			return
		}
	}
}
