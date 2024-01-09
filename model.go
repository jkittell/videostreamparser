package main

import (
	"github.com/google/uuid"
	"time"
)

type Result struct {
	Id       uuid.UUID `bson:"_id" json:"id"`
	URL      string    `bson:"url" json:"url"`
	Segments []Segment `bson:"segments,omitempty" json:"segments"`
}

type Job struct {
	Id          uuid.UUID `bson:"_id" json:"id"`
	URL         string    `bson:"url" json:"url"`
	Status      string    `bson:"status" json:"status"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	CompletedAt time.Time `bson:"completed_at" json:"completed_at"`
	Error       string    `bson:"error" json:"error"`
}

type Segment struct {
	PlaylistURL    string `bson:"playlist_url" json:"playlist_url"`
	StreamName     string `bson:"stream_name" json:"stream_name"`
	StreamURL      string `bson:"stream_url" json:"stream_url"`
	SegmentName    string `bson:"segment_name" json:"segment_name"`
	SegmentURL     string `bson:"segment_url" json:"segment_url"`
	ByteRangeStart int    `bson:"byte_range_start" json:"byte_range_start"`
	ByteRangeSize  int    `bson:"byte_range_size" json:"byte_range_size"`
}
