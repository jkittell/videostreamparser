package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func send(results chan Payload) {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"q.segments.response", // name
		false,                 // durable
		false,                 // delete when unused
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // arguments
	)
	failOnError(err, "Failed to declare a queue")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for res := range results {
		var buffer bytes.Buffer
		encoder := gob.NewEncoder(&buffer)
		if err := encoder.Encode(res); err != nil {
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
		if err != nil {
			log.Printf("[ %s ] failed to publish a message: %s", res.Id, err)
		}
		log.Printf("[ %s ] [<<] sent segments", res.Id)
	}
}
