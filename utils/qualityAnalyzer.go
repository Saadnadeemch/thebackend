package util

import (
	"log"
	"strings"
)

var qualityOrder = []string{
	"4k",
	"2k",
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
	"2k":    "bv*[height<=2160]+ba/b[height<=2160]",
	"4k":    "bv*[height<=4320]+ba/b[height<=4320]",
}

func CheckAndPickFormat(requestedQuality string) (string, string) {
	log.Printf("[QualityAnalyzer] Requested quality=%s", requestedQuality)

	start := -1
	for i, q := range qualityOrder {
		if q == requestedQuality {
			start = i
			break
		}
	}

	// Unknown quality → safest fallback
	if start == -1 {
		log.Printf("[QualityAnalyzer] Unknown quality -> fallback best")
		return "bv*+ba/b", "fallback_best"
	}

	var formats []string

	// Build fallback chain (requested → lower → lowest)
	for i := start; i < len(qualityOrder); i++ {
		q := qualityOrder[i]
		formats = append(formats, qualityFormatMap[q])
	}

	// Absolute final fallback
	formats = append(formats, "bv*+ba/b")

	finalFormat := strings.Join(formats, "/")

	log.Printf("[QualityAnalyzer] Selected format chain: %s", finalFormat)
	return finalFormat, "matched"
}
