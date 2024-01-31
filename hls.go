package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/jkittell/data/api/client"
	"github.com/jkittell/data/structures"
	"github.com/jkittell/toolbox"
	"log"
	"regexp"
	"strconv"
	"strings"
)

func decodeVariant(masterPlaylistURL, variantName, variantURL string, results *structures.Array[Segment]) error {
	//_, playlist, err := toolbox.SendRequest(toolbox.GET, variantURL, "", nil)
	playlist, err := client.Get(variantURL, nil, nil)
	if err != nil {
		return err
	}

	// store byte range then continue to next line for Segment
	var ByteRangeStart int
	var ByteRangeSize int

	segmentFormats := []string{".ts", ".fmp4", ".cmfv", ".cmfa", ".aac", ".ac3", ".ec3", ".webvtt"}
	scanner := bufio.NewScanner(bytes.NewReader(playlist))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "#EXT-X-BYTERANGE") {
			// #EXT-X-BYTERANGE:44744@2304880
			// -H "Range: bytes=0-1023"
			// parse byte range here
			byteRangeValues := strings.Split(line, ":")
			if len(byteRangeValues) != 2 {
				return err
			}
			byteRange := strings.Split(byteRangeValues[1], "@")
			startNumber, err := strconv.Atoi(byteRange[1])
			if err != nil {
				return err
			}
			sizeNumber, err := strconv.Atoi(byteRange[0])
			if err != nil {
				return err
			}

			ByteRangeStart = startNumber
			ByteRangeSize = sizeNumber
			continue
		} else {
			ByteRangeStart = -1
			ByteRangeSize = -1
		}

		for _, format := range segmentFormats {
			var match bool
			if strings.HasPrefix(line, "MISSING_") {
				continue
			}
			if strings.Contains(line, format) {
				match = true
				if match {
					if strings.Contains(line, "#EXT-X-MAP:URI=") {
						re := regexp.MustCompile(`"[^"]+"`)
						initSegment := re.FindString(line)
						if initSegment != "" {
							SegmentName := strings.Trim(initSegment, "\"")
							var SegmentURL string
							if !strings.Contains(SegmentName, "http") {
								baseURL := toolbox.BaseURL(variantURL)
								SegmentURL = fmt.Sprintf("%s/%s", baseURL, SegmentName)
							} else {
								SegmentURL = SegmentName
							}

							result := Segment{
								PlaylistURL:    masterPlaylistURL,
								StreamName:     variantName,
								StreamURL:      variantURL,
								SegmentName:    SegmentName,
								SegmentURL:     SegmentURL,
								ByteRangeStart: ByteRangeStart,
								ByteRangeSize:  ByteRangeSize,
							}
							results.Push(result)
						} else {
							return err
						}
					} else {
						SegmentName := line
						var SegmentURL string
						if !strings.Contains(SegmentName, "http") {
							baseURL := toolbox.BaseURL(variantURL)
							SegmentURL = fmt.Sprintf("%s/%s", baseURL, SegmentName)
						} else {
							SegmentURL = SegmentName
						}
						result := Segment{
							PlaylistURL:    masterPlaylistURL,
							StreamName:     variantName,
							StreamURL:      variantURL,
							SegmentName:    SegmentName,
							SegmentURL:     SegmentURL,
							ByteRangeStart: ByteRangeStart,
							ByteRangeSize:  ByteRangeSize,
						}
						results.Push(result)
					}
				}
			}
		}
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	return nil
}

func decodeMaster(url string) (map[string]string, error) {
	streams := make(map[string]string)
	playlist, err := client.Get(url, nil, nil)
	if err != nil {
		log.Println(err)
		return streams, err
	}

	baseURL := toolbox.BaseURL(url)
	scanner := bufio.NewScanner(bytes.NewReader(playlist))
	for scanner.Scan() {
		var streamURL string
		line := scanner.Text()
		if !strings.Contains(line, "#EXT") && strings.Contains(line, "m3u8") {
			if !strings.Contains(line, "http") {
				streamURL = fmt.Sprintf("%s/%s", baseURL, line)
			} else {
				streamURL = line
			}
			streams[line] = streamURL
		} else if strings.Contains(line, "#EXT-X-I-FRAME-STREAM-INF") || strings.Contains(line, "#EXT-X-MEDIA") {
			regEx := regexp.MustCompile("URI=\"(.*?)\"")
			match := regEx.MatchString(line)
			if match {
				s1 := regEx.FindString(line)
				_, s2, _ := strings.Cut(s1, "=")
				s3 := strings.Trim(s2, "\"")
				URI := s3
				if !strings.Contains(line, "http") {
					streamURL = fmt.Sprintf("%s/%s", baseURL, URI)
				} else {
					streamURL = URI
				}
				streams[URI] = streamURL
			}
		}
	}
	return streams, err
}

func parseHLS(url string) (*structures.Array[Segment], error) {
	results := structures.NewArray[Segment]()
	variants, err := decodeMaster(url)
	if err != nil {
		return results, err
	}

	if len(variants) > 0 {
		for variantName, variantURL := range variants {
			err = decodeVariant(url, variantName, variantURL, results)
			if err != nil {
				return results, err
			}
		}
	} else {
		err = decodeVariant(url, "", url, results)
	}
	return results, err
}
