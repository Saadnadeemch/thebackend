package util

import (
	"fmt"
)

func GetFragmentsWithConnection(activeUsers int, quality string) (string, string) {
	concurrentFragments, connections := 4, 4
	chunkSize := "2M"

	switch quality {
	case "144p", "240p":
		chunkSize = "1M"
	case "360p", "480p":
		chunkSize = "2M"
	case "720p", "1080p":
		chunkSize = "5M"
	case "1440p":
		chunkSize = "10M"
	case "2k", "4k":
		chunkSize = "50M"
	}

	// adjust concurrency based on users
	if activeUsers == 1 {
		concurrentFragments, connections = 8, 8
	} else if activeUsers <= 5 {
		concurrentFragments, connections = 6, 6
	} else {
		concurrentFragments, connections = 4, 4
	}

	fragArg := fmt.Sprintf("--concurrent-fragments=%d", concurrentFragments)
	dlArg := fmt.Sprintf("aria2c:-x%d -s%d -k%s", connections, connections, chunkSize)
	return fragArg, dlArg
}
