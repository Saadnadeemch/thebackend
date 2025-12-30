package util

import (
	"log"
)

var qualityFormatMap = map[string]string{
	"144p":  `bv*[height<=160][height>=120]+ba/b[height<=160][height>=120]/bv*+ba/b`,
	"240p":  `bv*[height<=260][height>=200]+ba/b[height<=260][height>=200]/bv*+ba/b`,
	"360p":  `bv*[height<=380][height>=320]+ba/b[height<=380][height>=320]/bv*+ba/b`,
	"480p":  `bv*[height<=510][height>=420]+ba/b[height<=510][height>=420]/bv*+ba/b`,
	"720p":  `bv*[height<=720][height>=480]+ba/b[height<=720][height>=480]/bv*+ba/b`,
	"1080p": `bv*[height<=1080][height>=720]+ba/b[height<=1080][height>=720]/bv*+ba/b`,
	"1440p": `bv*[height<=1440][height>=1080]+ba/b[height<=1440][height>=1080]/bv*+ba/b`,
	"2k":    `bv*[height<=2160][height>=1440]+ba/b[height<=2160][height>=1440]/bv*+ba/b`,
	"4k":    `bv*[height<=4320][height>=2160]+ba/b[height<=4320][height>=2160]/bv*+ba/b`,
}

func CheckAndPickFormat(videoURL, requestedQuality string) (string, string) {
	log.Printf("[QualityAnalyzer]  Requested quality=%s | URL=%s", requestedQuality, videoURL)

	if selector, ok := qualityFormatMap[requestedQuality]; ok {
		log.Printf("[QualityAnalyzer]  Matched rule for %s", requestedQuality)
		return selector, "matched"
	}

	// Unknown quality fallback
	log.Printf("[QualitQualityAnalyzerySelector]  Unknown quality=%s -> fallback best available", requestedQuality)
	return "bv*+ba/b", "fallback_best"
}
