// app_news.go - Minecraft news and launcher release notes Wails bindings.

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// MinecraftNewsItem represents a news or patch note article from Mojang.
type MinecraftNewsItem struct {
	Title       string `json:"title"`
	Tag         string `json:"tag"`
	Date        string `json:"date"`
	Text        string `json:"text"`
	Image       string `json:"image"`
	ReadMoreURL string `json:"readMoreUrl"`
}

// GetMinecraftNews fetches official patch notes and announcements from Mojang.
func (a *App) GetMinecraftNews() ([]MinecraftNewsItem, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 6*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://launchercontent.mojang.com/v2/news.json", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []MinecraftNewsItem{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []MinecraftNewsItem{}, nil
	}

	var payload struct {
		Entries []struct {
			Title         string `json:"title"`
			Tag           string `json:"tag"`
			Category      string `json:"category"`
			Date          string `json:"date"`
			Text          string `json:"text"`
			NewsPageImage struct {
				URL string `json:"url"`
			} `json:"newsPageImage"`
			PlayPageImage struct {
				URL string `json:"url"`
			} `json:"playPageImage"`
			ReadMoreLink string `json:"readMoreLink"`
		} `json:"entries"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return []MinecraftNewsItem{}, nil
	}

	var results []MinecraftNewsItem
	for i, entry := range payload.Entries {
		if i >= 6 {
			break
		}
		img := entry.NewsPageImage.URL
		if img == "" {
			img = entry.PlayPageImage.URL
		}
		if img != "" && !strings.HasPrefix(img, "http") {
			img = "https://launchercontent.mojang.com" + img
		}
		tag := entry.Tag
		if tag == "" {
			tag = entry.Category
		}
		results = append(results, MinecraftNewsItem{
			Title:       entry.Title,
			Tag:         tag,
			Date:        entry.Date,
			Text:        entry.Text,
			Image:       img,
			ReadMoreURL: entry.ReadMoreLink,
		})
	}

	return results, nil
}

// AetherReleaseNote represents a release note entry from Aether Launcher GitHub releases.
type AetherReleaseNote struct {
	TagName     string `json:"tagName"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"publishedAt"`
	HTMLURL     string `json:"htmlUrl"`
}

// GetAetherReleaseNotes fetches the latest GitHub release notes for Aether Launcher.
func (a *App) GetAetherReleaseNotes() ([]AetherReleaseNote, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 6*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/Aether-Launcher/Aether/releases?per_page=6", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Aether-Launcher")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []AetherReleaseNote{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []AetherReleaseNote{}, nil
	}

	var raw []struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
		HTMLURL     string `json:"html_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return []AetherReleaseNote{}, nil
	}

	var results []AetherReleaseNote
	for _, r := range raw {
		name := r.Name
		if name == "" {
			name = r.TagName
		}
		results = append(results, AetherReleaseNote{
			TagName:     r.TagName,
			Name:        name,
			Body:        r.Body,
			PublishedAt: r.PublishedAt,
			HTMLURL:     r.HTMLURL,
		})
	}
	return results, nil
}
