package util

import (
	"backend/models" // Replace 'your-project-path' with your actual module name
	"net/url"
	"strings"
)

type platformRule struct {
	reelPaths         []string
	videoPaths        []string
	defaultType       models.VideoType
	defaultConfidence models.Confidence
}

var hostToPlatform = map[string]string{
	"youtube.com":        "YouTube",
	"m.youtube.com":      "YouTube",
	"music.youtube.com":  "YouTube",
	"youtu.be":           "YouTube",
	"vimeo.com":          "Vimeo",
	"player.vimeo.com":   "Vimeo",
	"facebook.com":       "Facebook",
	"m.facebook.com":     "Facebook",
	"web.facebook.com":   "Facebook",
	"fb.watch":           "Facebook",
	"dailymotion.com":    "Dailymotion",
	"m.dailymotion.com":  "Dailymotion",
	"dai.ly":             "Dailymotion",
	"instagram.com":      "Instagram",
	"m.instagram.com":    "Instagram",
	"instagr.am":         "Instagram",
	"ig.me":              "Instagram",
	"twitter.com":        "Twitter",
	"mobile.twitter.com": "Twitter",
	"x.com":              "Twitter",
	"tiktok.com":         "TikTok",
	"m.tiktok.com":       "TikTok",
	"vm.tiktok.com":      "TikTok",
	"vt.tiktok.com":      "TikTok",
	"t.tiktok.com":       "TikTok",
	"twitch.tv":          "Twitch",
	"m.twitch.tv":        "Twitch",
	"clips.twitch.tv":    "Twitch",
	"reddit.com":         "Reddit",
	"m.reddit.com":       "Reddit",
	"redd.it":            "Reddit",
	"v.redd.it":          "Reddit",
	"pinterest.com":      "Pinterest",
	"pin.it":             "Pinterest",
	"linkedin.com":       "LinkedIn",
	"m.linkedin.com":     "LinkedIn",
	"vk.com":             "VK",
	"m.vk.com":           "VK",
	"vkontakte.ru":       "VK",
	"rutube.ru":          "Rutube",
	"m.rutube.ru":        "Rutube",
	"ok.ru":              "OK.ru",
	"m.ok.ru":            "OK.ru",
	"peertube.com":       "PeerTube",
	"peertube.cpy.re":    "PeerTube",
	"video.cpy.re":       "PeerTube",
	"bandcamp.com":       "Bandcamp",
	"bitchute.com":       "BitChute",
	"m.bitchute.com":     "BitChute",
	"rumble.com":         "Rumble",
	"vine.co":            "Vine",
	"ted.com":            "TED",
	"streamable.com":     "Streamable",
	"bilibili.tv":        "Bilibili",
	"m.bilibili.tv":      "Bilibili",
	"bilibili.com":       "Bilibili",
	"mastodon.social":    "Mastodon",
}

var platformRules = map[string]platformRule{
	"YouTube": {
		reelPaths:         []string{"/shorts"},
		videoPaths:        []string{"/watch", "/embed", "/live"},
		defaultType:       models.VideoTypeVideo,
		defaultConfidence: models.ConfidenceMedium,
	},
	"Instagram": {
		reelPaths:         []string{"/reel", "/reels"},
		videoPaths:        []string{"/tv", "/p"},
		defaultType:       models.VideoTypeReel,
		defaultConfidence: models.ConfidenceMedium,
	},
	"Facebook": {
		reelPaths:         []string{"/reel", "/reels"},
		videoPaths:        []string{"/watch", "/video", "/videos"},
		defaultType:       models.VideoTypeVideo,
		defaultConfidence: models.ConfidenceMedium,
	},
	"TikTok": {
		reelPaths:         []string{},
		videoPaths:        []string{},
		defaultType:       models.VideoTypeReel,
		defaultConfidence: models.ConfidenceHigh,
	},
	"Twitter": {
		reelPaths:         []string{"/status"},
		videoPaths:        []string{},
		defaultType:       models.VideoTypeReel,
		defaultConfidence: models.ConfidenceHigh,
	},
	"Twitch": {
		reelPaths:         []string{"/clip", "/clips"},
		videoPaths:        []string{"/videos"},
		defaultType:       models.VideoTypeVideo,
		defaultConfidence: models.ConfidenceMedium,
	},
	"Reddit": {
		reelPaths:         []string{},
		videoPaths:        []string{},
		defaultType:       models.VideoTypeReel,
		defaultConfidence: models.ConfidenceMedium,
	},
	"Dailymotion": {
		reelPaths:         []string{},
		videoPaths:        []string{"/video"},
		defaultType:       models.VideoTypeVideo,
		defaultConfidence: models.ConfidenceHigh,
	},
	"Vimeo": {
		reelPaths:         []string{},
		videoPaths:        []string{},
		defaultType:       models.VideoTypeVideo,
		defaultConfidence: models.ConfidenceHigh,
	},
	"Bilibili": {
		reelPaths:         []string{"/video/BV", "/video/AV"},
		videoPaths:        []string{"/video"},
		defaultType:       models.VideoTypeVideo,
		defaultConfidence: models.ConfidenceHigh,
	},
	"Pinterest": {
		reelPaths:         []string{},
		videoPaths:        []string{},
		defaultType:       models.VideoTypeReel,
		defaultConfidence: models.ConfidenceMedium,
	},
	"LinkedIn": {
		reelPaths:         []string{},
		videoPaths:        []string{"/posts", "/feed"},
		defaultType:       models.VideoTypeVideo,
		defaultConfidence: models.ConfidenceMedium,
	},
	"Rumble": {
		reelPaths:         []string{},
		videoPaths:        []string{},
		defaultType:       models.VideoTypeVideo,
		defaultConfidence: models.ConfidenceHigh,
	},
	"Streamable": {
		reelPaths:         []string{},
		videoPaths:        []string{},
		defaultType:       models.VideoTypeVideo,
		defaultConfidence: models.ConfidenceHigh,
	},
	"TED": {
		reelPaths:         []string{},
		videoPaths:        []string{"/talks"},
		defaultType:       models.VideoTypeVideo,
		defaultConfidence: models.ConfidenceHigh,
	},
}

func DetectPlatform(inputURL string) models.PlatformInfo {
	parsed, err := url.Parse(inputURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return models.PlatformInfo{
			Platform:   "Unknown",
			VideoType:  models.VideoTypeVideo,
			Confidence: models.ConfidenceLow,
		}
	}

	host := strings.ToLower(parsed.Host)
	host = strings.TrimPrefix(host, "www.")

	platform, exists := hostToPlatform[host]
	if !exists {
		return models.PlatformInfo{
			Platform:   "Unknown",
			VideoType:  models.VideoTypeVideo,
			Confidence: models.ConfidenceLow,
		}
	}

	rule, hasRule := platformRules[platform]
	if !hasRule {
		return models.PlatformInfo{
			Platform:   platform,
			VideoType:  models.VideoTypeVideo,
			Confidence: models.ConfidenceLow,
		}
	}

	path := strings.ToLower(parsed.Path)

	for _, reelPath := range rule.reelPaths {
		if strings.Contains(path, strings.ToLower(reelPath)) {
			return models.PlatformInfo{
				Platform:   platform,
				VideoType:  models.VideoTypeReel,
				Confidence: models.ConfidenceHigh,
			}
		}
	}

	for _, videoPath := range rule.videoPaths {
		if strings.Contains(path, strings.ToLower(videoPath)) {
			return models.PlatformInfo{
				Platform:   platform,
				VideoType:  models.VideoTypeVideo,
				Confidence: models.ConfidenceHigh,
			}
		}
	}

	return models.PlatformInfo{
		Platform:   platform,
		VideoType:  rule.defaultType,
		Confidence: rule.defaultConfidence,
	}
}
