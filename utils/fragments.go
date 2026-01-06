package util

import "fmt"

func GetFragmentsWithConnection(activeUsers int, quality string) string {

	concurrentFragments := 3

	switch {
	case activeUsers <= 1:
		concurrentFragments = 4
	case activeUsers <= 3:
		concurrentFragments = 3
	default:
		concurrentFragments = 2
	}

	switch quality {
	case "144p", "240p":
		if concurrentFragments > 3 {
			concurrentFragments = 3
		}
	case "360p", "480p":
		if concurrentFragments > 3 {
			concurrentFragments = 3
		}
	case "720p", "1080p":
		if concurrentFragments > 4 {
			concurrentFragments = 4
		}
	case "1440p", "2k", "4k":
		if concurrentFragments > 4 {
			concurrentFragments = 4
		}
	}

	return fmt.Sprintf("--concurrent-fragments=%d", concurrentFragments)
}
