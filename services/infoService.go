package services

import (
	"backend/models"
	ytdlp "backend/yt-dlp"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

func GetVideoInfoService(videoURL string, platform string, method string) (*models.VideoInfo, error) {
	if method == "direct-download" {
		log.Println("[InfoService] Method is direct-download → using yt-dlp only")
		info, err := ytdlp.GetDirectInfoFromYTDLP(videoURL)
		if err == nil {
			info.Source = "yt-dlp"
			return info, nil
		}
		return nil, errors.New("yt-dlp failed for direct-download")
	}

	// Try Iframely first
	log.Println("[InfoService] Using Iframly")

	info, err := getInfoFromIframly(videoURL)
	if err == nil {
		info.Source = "iframely"
		return info, nil
	}

	log.Println("[InfoService] Using Yt-DLP ")

	// Fallback to yt-dlp
	info, err = ytdlp.GetInfoFromYTDLP(videoURL)
	if err == nil {
		info.Source = "yt-dlp"
		return info, nil
	}

	return nil, errors.New("all sources failed to fetch video info")
}

func GetAudioInfo(videoURL string) (*models.VideoInfo, error) {
	info, err := getInfoFromIframly(videoURL)
	if err != nil {
		return nil, err
	}
	info.Source = "iframely"
	return info, nil
}

func getInfoFromIframly(videoURL string) (*models.VideoInfo, error) {
	apiURL := "http://localhost:8061/iframely?url=" + videoURL

	// Hard 10 seconds timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, fmt.Errorf("iframely error: %v", err)
	}
	defer resp.Body.Close()

	var raw struct {
		Meta struct {
			Title       string `json:"title"`
			Author      string `json:"author"`
			AuthorURL   string `json:"author_url"`
			Site        string `json:"site"`
			Description string `json:"description"`
			Canonical   string `json:"canonical"`
		} `json:"meta"`
		Links []struct {
			Href string   `json:"href"`
			Rel  []string `json:"rel"`
			Type string   `json:"type"`
		} `json:"links"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	// Pick first thumbnail
	var thumb string
	for _, link := range raw.Links {
		for _, rel := range link.Rel {
			if rel == "thumbnail" {
				thumb = link.Href
				break
			}
		}
		if thumb != "" {
			break
		}
	}

	return &models.VideoInfo{
		Title:     raw.Meta.Title,
		Uploader:  raw.Meta.Author,
		Thumbnail: thumb,
		VideoPage: videoURL,
		Views:     0,
	}, nil
}
