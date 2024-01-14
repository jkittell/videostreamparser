package main

import (
	"errors"
	"fmt"
	"github.com/jkittell/data/structures"
	"strings"
)

type Segment struct {
	PlaylistURL    string `json:"playlist_url" bson:"playlist_url"`
	StreamName     string `json:"stream_name" bson:"stream_name"`
	StreamURL      string `json:"stream_url" bson:"stream_url"`
	SegmentName    string `json:"segment_name" bson:"segment_name"`
	SegmentURL     string `json:"segment_url" bson:"segment_url"`
	ByteRangeStart int    `json:"byte_range_start" bson:"byte_range_start"`
	ByteRangeSize  int    `json:"byte_range_size" bson:"byte_range_size"`
}

func getSegments(url string) (*structures.Array[Segment], error) {
	if strings.Contains(url, "m3u8") {
		return parseHLS(url)
	} else if strings.Contains(url, "mpd") {
		return parseDASH(url)
	} else {
		return structures.NewArray[Segment](), errors.New(fmt.Sprintf("unable to parse %s", url))
	}
}

func GetSegments(url string) (*structures.Array[Segment], error) {
	segments, err := getSegments(url)
	if err != nil {
		return segments, err
	}

	return segments, nil
}
