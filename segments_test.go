package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"testing"
	"time"
)

var url = "http://devimages.apple.com/iphone/samples/bipbop/bipbopall.m3u8"

func TestSegments(t *testing.T) {
	done := make(chan bool)
	go request()
	go response()
	<-done
}

func request() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"q.segments.in", // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	failOnError(err, "Failed to declare a queue")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for range time.Tick(time.Second) {
		payload := Payload{
			Id:       uuid.New(),
			URL:      url,
			Segments: nil,
		}
		var buffer bytes.Buffer
		encoder := gob.NewEncoder(&buffer)
		if err := encoder.Encode(payload); err != nil {
			log.Println(err)
			continue
		}
		err = ch.PublishWithContext(ctx,
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: "application/x-gob",
				Body:        buffer.Bytes(),
			})
		failOnError(err, "Failed to publish a message")
		log.Printf("[<<] Sent %s\n", payload.Id.String())
	}

}

func response() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"q.segments.out", // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	failOnError(err, "Failed to declare a queue")

	messages, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	done := make(chan bool)

	go func() {
		for d := range messages {
			dec := gob.NewDecoder(bytes.NewReader(d.Body))
			var p Payload
			err = dec.Decode(&p)
			if err != nil {
				log.Fatal("decode:", err)
			}
			log.Printf("[>>] Received: %s\n", p.Id.String())
			segments, err := GetSegments(p.URL)
			if err != nil {
				log.Println(err)
				d.Nack(false, true)
			}
			p.Segments = segments.ToSlice()
			log.Printf("got %d segments for %s\n", len(p.Segments), p.Id)
			d.Ack(false)
		}
	}()

	<-done
}
