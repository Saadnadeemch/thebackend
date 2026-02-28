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

func GetVideoInfoService(videoURL string, VideoType string) (*models.VideoInfo, error) {

	log.Println("[InfoService] Using Iframely")

	info, err := getInfoFromIframly(videoURL)
	if err == nil && info.Title != "" {
		info.Source = "iframely"
		return info, nil
	}

	log.Println("[InfoService] Using Yt-DLP")

	info, err = ytdlp.GetVideoInfoFromYTDLP(videoURL)
	if err == nil {
		info.Source = "yt-dlp"
		return info, nil
	}

	return nil, errors.New("all sources failed to fetch video info")
}

func getInfoFromIframly(videoURL string) (*models.VideoInfo, error) {
	apiURL := "http://localhost:8061/iframely?url=" + videoURL

	client := &http.Client{
		Timeout: 7 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("iframely request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iframely bad status: %d", resp.StatusCode)
	}

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

	if raw.Meta.Title == "" {
		return nil, errors.New("iframely returned empty title")
	}

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
