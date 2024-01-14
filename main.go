package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jkittell/data/database"
	"github.com/wagslane/go-rabbitmq"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Job struct {
	Id          uuid.UUID `json:"id" bson:"id"`
	StartedAt   time.Time `json:"started_at" bson:"started_at"`
	CompletedAt time.Time `json:"completed_at" bson:"completed_at"`
	URL         string    `json:"URL" bson:"url"`
	Segments    []Segment `json:"segments" bson:"segments"`
}

func main() {
	db, err := database.NewMongoDB[Job]()
	if err != nil {
		log.Fatal(err)
	}
	conn, err := rabbitmq.NewConn(
		os.Getenv("RABBITMQ_URL"),
		rabbitmq.WithConnectionOptionsLogging,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	consumer, err := rabbitmq.NewConsumer(
		conn,
		func(d rabbitmq.Delivery) rabbitmq.Action {
			var job Job
			err = json.Unmarshal(d.Body, &job)
			if err != nil {
				log.Println(err)
				return rabbitmq.NackDiscard
			}

			job.StartedAt = time.Now()
			segments, err := GetSegments(job.URL)
			if err != nil {
				log.Println(err)
				return rabbitmq.NackDiscard
			}

			job.CompletedAt = time.Now()
			job.Segments = segments.ToSlice()
			err = db.Insert(context.TODO(), "segments", job)
			if err != nil {
				log.Println(err)
				return rabbitmq.NackDiscard
			}

			return rabbitmq.Ack
		},
		os.Getenv("RABBITMQ_QUEUE"),
		rabbitmq.WithConsumerOptionsRoutingKey(os.Getenv("RABBITMQ_ROUTING_KEY")),
		rabbitmq.WithConsumerOptionsExchangeName(os.Getenv("RABBITMQ_EXCHANGE_NAME")),
		rabbitmq.WithConsumerOptionsExchangeDeclare,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	// block main thread - wait for shutdown signal
	sigs := make(chan os.Signal)
	done := make(chan bool)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	<-done
}
