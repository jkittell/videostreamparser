package main

import (
	"errors"
	"fmt"
	"github.com/jkittell/data/structures"
	"strings"
)

func GetSegments(url string) (*structures.Array[Segment], error) {
	if strings.Contains(url, "m3u8") {
		return parseHLS(url)
	} else if strings.Contains(url, "mpd") {
		return parseDASH(url)
	} else {
		return structures.NewArray[Segment](), errors.New(fmt.Sprintf("unable to parse %s", url))
	}
}
