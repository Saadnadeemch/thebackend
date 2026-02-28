package util

import (
	"log"
	"strings"
)

var qualityOrder = []string{
	"1440p",
	"1080p",
	"720p",
	"480p",
	"360p",
	"240p",
	"144p",
}

var qualityFormatMap = map[string]string{
	"144p":  "bv*[height<=144]+ba/b[height<=144]",
	"240p":  "bv*[height<=240]+ba/b[height<=240]",
	"360p":  "bv*[height<=360]+ba/b[height<=360]",
	"480p":  "bv*[height<=480]+ba/b[height<=480]",
	"720p":  "bv*[height<=720]+ba/b[height<=720]",
	"1080p": "bv*[height<=1080]+ba/b[height<=1080]",
	"1440p": "bv*[height<=1440]+ba/b[height<=1440]",
}

func CheckAndPickFormat(requestedQuality string, platfromInfo string) (string, string) {
	log.Printf("[QualityAnalyzer] Requested quality=%s", requestedQuality)

	start := -1
	for i, q := range qualityOrder {
		if q == requestedQuality {
			start = i
			break
		}
	}

	if start == -1 {
		log.Printf("[QualityAnalyzer] Unknown quality -> fallback best")
		return "bv*+ba/b", "fallback_best"
	}

	var formats []string

	for i := start; i < len(qualityOrder); i++ {
		q := qualityOrder[i]
		formats = append(formats, qualityFormatMap[q])
	}

	formats = append(formats, "bv*+ba/b")

	finalFormat := strings.Join(formats, "/")

	log.Printf("[QualityAnalyzer] Selected format chain: %s", finalFormat)
	return finalFormat, "matched"
}

func GetFragmentsByQuality(quality string) string {

	switch {
	case strings.Contains(quality, "144"):
		return "2"
	case strings.Contains(quality, "240"):
		return "2"
	case strings.Contains(quality, "360"):
		return "4"
	case strings.Contains(quality, "480"):
		return "6"
	case strings.Contains(quality, "720"):
		return "8"
	case strings.Contains(quality, "1080"):
		return "12"
	case strings.Contains(quality, "1440"):
		return "16"
	case strings.Contains(quality, "2160"):
		return "16"
	default:
		return "6"
	}
}
