package main

import (
	"github.com/google/uuid"
)

type Payload struct {
	Id       uuid.UUID
	URL      string
	Segments []Segment
}
