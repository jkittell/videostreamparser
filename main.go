package main

import (
	"log"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	done := make(chan bool)
	results := make(chan Payload)
	go receive(results)
	go send(results)
	<-done
}
