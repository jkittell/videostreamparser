package main

import (
	"fmt"
	"github.com/jkittell/data/structures"
	"github.com/jkittell/toolbox"
	"github.com/unki2aut/go-mpd"
	"regexp"
)

/*
<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns="urn:mpeg:dash:schema:mpd:2011" xsi:schemaLocation="urn:mpeg:dash:schema:mpd:2011 http://standards.iso.org/ittf/PubliclyAvailableStandards/MPEG-DASH_schema_files/DASH-MPD.xsd" availabilityStartTime="2023-03-23T17:20:36Z" type="dynamic" mediaPresentationDuration="PT0H15M10S" publishTime="2023-03-23T17:35:17Z" maxSegmentDuration="PT10S" minBufferTime="PT10S" profiles="urn:scte:dash:2015#ts">
    <Period id="1" start="PT0S">
        <AdaptationSet id="1" mimeType="video/mp2t" segmentAlignment="true" bitStreamSwitching="true" lang="und" maxWidth="960" maxHeight="540" maxFrameRate="29">
            <Representation id="Stream1-1" audioSamplingRate="48000" codecs="avc1.64001F,ac-3,mp4a.40.2" width="960" height="540" frameRate="29" sar="16:9" startWithSAP="1" bandwidth="3199008">
                <SegmentTemplate timescale="90000" media="CCURStream_$RepresentationID$_$Number$.ts?ccur_ts_audio_Stream=Stream1&amp;ccur_ts_audio_track=10&amp;ccur_ts_audio_Stream=Stream1&amp;ccur_ts_audio_track=11" startNumber="167791408" presentationTimeOffset="1744652613072" duration="900000"/>
            </Representation>
        </AdaptationSet>
    </Period>
</MPD>

*/

type segmentTemplate struct {
	segmentDuration  uint64
	timescale        uint64
	startNumber      uint64
	manifestDuration float64
	baseURL          string
	representationId string
	media            string
	playlistURL      string
	streamName       string
}

type segmentTimeline struct {
	baseURL          string
	representationId string
	media            string
	playlistURL      string
}

// calculateDashSegmentTimestamp is used to calculate the timestamp values for the segment in the dash segment timeline
func calculateDashSegmentTimestamp(timestampOfFirstSegment uint64, segmentDuration uint64, segmentRepeat int64) []uint64 {
	var timestamps []uint64
	var timestamp uint64
	var i int64

	// Loop the number of times indicated by the segment repeat value
	if segmentRepeat > 0 {
		for i = 0; i < segmentRepeat; i++ {
			// If it's the first loop use the timestamp of the first segment
			// otherwise increment the timestamp by the segment duration
			if i > 0 {
				timestamp = timestamp + segmentDuration
			} else {
				timestamp = timestampOfFirstSegment
			}
			timestamps = append(timestamps, timestamp)
		}
	} else {
		timestamps = append(timestamps, timestampOfFirstSegment)
	}

	return timestamps
}

func getSegmentsFromSegmentTimeline(dashSegmentTimestamps []uint64, segmentTimeline segmentTimeline, results *structures.Array[Segment]) {
	for _, timestamp := range dashSegmentTimestamps {
		var representationRegex = `\$RepresentationID\$`
		var timeRegex = `\$Time\$`

		var segmentName string
		var r = regexp.MustCompile(representationRegex)
		segmentName = r.ReplaceAllString(segmentTimeline.media, segmentTimeline.representationId)

		var n = regexp.MustCompile(timeRegex)
		segmentName = n.ReplaceAllString(segmentName, fmt.Sprint(timestamp))

		result := Segment{
			PlaylistURL:    segmentTimeline.playlistURL,
			StreamName:     segmentTimeline.representationId,
			StreamURL:      "",
			SegmentName:    segmentName,
			SegmentURL:     fmt.Sprintf("%s/%s", segmentTimeline.baseURL, segmentName),
			ByteRangeStart: -1,
			ByteRangeSize:  -1,
		}
		results.Push(result)
	}
}

// TODO init
func getSegmentsFromSegmentTemplate(segmentTemplate segmentTemplate, results *structures.Array[Segment]) {
	// get the segment size
	// duration="900000" / timescale="90000"
	// so 10 second segments
	segmentSize := segmentTemplate.segmentDuration / segmentTemplate.timescale

	// divide the media presentation duration / segment size
	// to get the number of segments
	numberOfSegments := uint64(segmentTemplate.manifestDuration) / segmentSize

	// start number - N where N is number of segments to get the last
	// segment in the window then increment N times
	N := segmentTemplate.startNumber + numberOfSegments

	for i := segmentTemplate.startNumber; i < N; i++ {
		segmentNumber := fmt.Sprintf("%d", i)
		var representationRegex = `\$RepresentationID\$`
		var numberRegex = `\$Number\$`

		var segmentName string
		var r = regexp.MustCompile(representationRegex)
		segmentName = r.ReplaceAllString(segmentTemplate.media, segmentTemplate.representationId)

		var n = regexp.MustCompile(numberRegex)
		segmentName = n.ReplaceAllString(segmentName, segmentNumber)

		result := Segment{
			PlaylistURL:    segmentTemplate.playlistURL,
			StreamName:     segmentTemplate.streamName,
			StreamURL:      "",
			SegmentName:    segmentName,
			SegmentURL:     fmt.Sprintf("%s/%s", segmentTemplate.baseURL, segmentName),
			ByteRangeStart: -1,
			ByteRangeSize:  -1,
		}
		results.Push(result)
	}
}

func getManifest(url string) (*mpd.MPD, error) {
	dashManifest := new(mpd.MPD)
	_, manifestFile, err := toolbox.SendRequest(toolbox.GET, url, "", nil)
	if err != nil {
		return dashManifest, err
	}

	err = dashManifest.Decode(manifestFile)
	if err != nil {
		return dashManifest, err
	}
	return dashManifest, nil
}

func parseDASH(url string) (*structures.Array[Segment], error) {
	results := structures.NewArray[Segment]()
	representations := make(map[string]string)

	var segmentDuration uint64
	var timescale uint64
	var startNumber uint64
	var baseURL string
	var representationId string
	var media string

	baseURL = toolbox.BaseURL(url)

	dashManifest, err := getManifest(url)
	if err != nil {
		return results, err
	}

	var manifestDuration float64

	if dashManifest.Type != nil {
		if *dashManifest.Type == "static" {
			d, err := dashManifest.MediaPresentationDuration.ToSeconds()
			if err != nil {
				return results, err
			}
			manifestDuration = d
		} else if *dashManifest.Type == "dynamic" {
			// 1/18/24 container was crashing because time shift buffer depth was not in dash ts manifest
			if dashManifest.TimeShiftBufferDepth != nil {
				d, err := dashManifest.TimeShiftBufferDepth.ToSeconds()
				if err != nil {
					return results, err
				}
				manifestDuration = d
			}
		}
	}

	for _, period := range dashManifest.Period {
		for _, set := range period.AdaptationSets {
			for _, rep := range set.Representations {
				representationId = *rep.ID
				representations[representationId] = url
				timescale = *rep.SegmentTemplate.Timescale
				media = *rep.SegmentTemplate.Media

				if rep.SegmentTemplate.StartNumber != nil {
					startNumber = *rep.SegmentTemplate.StartNumber
					segmentDuration = *rep.SegmentTemplate.Duration
					template := segmentTemplate{
						segmentDuration:  segmentDuration,
						timescale:        timescale,
						startNumber:      startNumber,
						manifestDuration: manifestDuration,
						baseURL:          baseURL,
						representationId: representationId,
						media:            media,
						playlistURL:      url,
						streamName:       representationId,
					}
					getSegmentsFromSegmentTemplate(template, results)
				} else {
					var dashSegmentTimestamps []uint64
					for _, timeline := range rep.SegmentTemplate.SegmentTimeline.S {
						timestampOfFirstSegment := *timeline.T
						segmentDuration = timeline.D
						var segmentRepeat int64
						if timeline.R != nil {
							segmentRepeat = *timeline.R
						} else {
							segmentRepeat = 0
						}
						timestamps := calculateDashSegmentTimestamp(timestampOfFirstSegment, segmentDuration, segmentRepeat)
						dashSegmentTimestamps = append(dashSegmentTimestamps, timestamps...)
					}
					timeline := segmentTimeline{
						baseURL:          baseURL,
						representationId: representationId,
						media:            media,
						playlistURL:      url,
					}
					getSegmentsFromSegmentTimeline(dashSegmentTimestamps, timeline, results)
				}
			}
		}
	}

	return results, nil
}
