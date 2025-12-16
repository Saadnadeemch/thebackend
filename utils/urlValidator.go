package util

import (
	"net/url"
	"strings"

	"backend/models"
)

// host → platform mapping
var hostToPlatform = map[string]string{
	// YouTube
	"youtube.com":       "YouTube",
	"m.youtube.com":     "YouTube",
	"music.youtube.com": "YouTube",
	"youtu.be":          "YouTube",

	// Vimeo
	"vimeo.com":        "Vimeo",
	"player.vimeo.com": "Vimeo",

	// Facebook
	"facebook.com":     "Facebook",
	"m.facebook.com":   "Facebook",
	"web.facebook.com": "Facebook",
	"fb.watch":         "Facebook",

	// Dailymotion
	"dailymotion.com":   "Dailymotion",
	"m.dailymotion.com": "Dailymotion",
	"dai.ly":            "Dailymotion",

	// Instagram
	"instagram.com":   "Instagram",
	"m.instagram.com": "Instagram",
	"instagr.am":      "Instagram",
	"ig.me":           "Instagram",

	// Twitter / X
	"twitter.com":        "Twitter",
	"mobile.twitter.com": "Twitter",
	"x.com":              "Twitter",

	// TikTok
	"tiktok.com":    "TikTok",
	"m.tiktok.com":  "TikTok",
	"vm.tiktok.com": "TikTok",
	"vt.tiktok.com": "TikTok",
	"t.tiktok.com":  "TikTok",

	// Twitch
	"twitch.tv":       "Twitch",
	"m.twitch.tv":     "Twitch",
	"clips.twitch.tv": "Twitch",

	// Reddit
	"reddit.com":   "Reddit",
	"m.reddit.com": "Reddit",
	"redd.it":      "Reddit",
	"v.redd.it":    "Reddit",

	// Pinterest
	"pinterest.com": "Pinterest",
	"pin.it":        "Pinterest",

	// LinkedIn
	"linkedin.com":     "LinkedIn",
	"www.linkedin.com": "LinkedIn",
	"m.linkedin.com":   "LinkedIn",

	// VK
	"vk.com":       "VK",
	"m.vk.com":     "VK",
	"vkontakte.ru": "VK",

	// Rutube
	"rutube.ru":   "Rutube",
	"m.rutube.ru": "Rutube",

	// OK.ru
	"ok.ru":   "OK.ru",
	"m.ok.ru": "OK.ru",

	// PeerTube
	"peertube.com":    "PeerTube",
	"peertube.cpy.re": "PeerTube",
	"video.cpy.re":    "PeerTube",

	// Bandcamp
	"bandcamp.com": "Bandcamp",

	// BitChute
	"bitchute.com":   "BitChute",
	"m.bitchute.com": "BitChute",

	// Rumble (still in map but not in supported list → default)
	"rumble.com": "Rumble",

	// Vine (legacy)
	"vine.co": "Vine",

	// TED
	"ted.com": "TED",

	// Streamable
	"streamable.com": "Streamable",

	// Bilibili
	"bilibili.tv":     "Bilibili",
	"www.bilibili.tv": "Bilibili",
	"m.bilibili.tv":   "Bilibili",
	"bilibili.com":    "Bilibili",

	// Mastodon
	"mastodon.social": "Mastodon",
}

// Platforms with separate audio/video streams
var separateAVPlatforms = map[string]bool{
	"YouTube":  true,
	"Reddit":   true,
	"Bilibili": true,
	"Rutube":   true,
	"TED":      true,
}

// Platforms that work best with streaming approach
var streamDownloadPlatforms = map[string]bool{
	"TikTok":      true,
	"Twitch":      true,
	"Dailymotion": true,
}

// Platforms that support direct file downloads
var directDownloadPlatforms = map[string]bool{
	"Facebook":   true,
	"Instagram":  true,
	"Vimeo":      true,
	"Twitter":    true,
	"Pinterest":  true,
	"LinkedIn":   true,
	"VK":         true,
	"OK.ru":      true,
	"PeerTube":   true,
	"Bandcamp":   true,
	"BitChute":   true,
	"Streamable": true,
	"Vine":       true,
	"Mastodon":   true,
}

// DetectPlatform validates URL, detects platform, and decides download method.
func DetectPlatform(inputURL string) models.PlatformInfo {
	parsed, err := url.Parse(inputURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return models.PlatformInfo{
			Platform:       "Invalid",
			IsSupported:    false,
			DownloadMethod: "default",
			Reason:         "Invalid URL",
		}
	}

	// Normalize host
	host := strings.ToLower(parsed.Host)
	host = strings.TrimPrefix(host, "www.")

	platform, exists := hostToPlatform[host]
	if !exists {
		return models.PlatformInfo{
			Platform:       "default",
			IsSupported:    false,
			DownloadMethod: "default",
			Reason:         "Platform not recognized or not supported",
		}
	}

	// Decide download method
	method := "default"
	switch {
	case separateAVPlatforms[platform]:
		method = "separate-av"
	case streamDownloadPlatforms[platform]:
		method = "stream-download"
	case directDownloadPlatforms[platform]:
		method = "direct-download"
	}

	return models.PlatformInfo{
		Platform:       platform,
		IsSupported:    true,
		DownloadMethod: method,
		Reason:         "",
	}
}
