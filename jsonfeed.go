package main

import (
	"time"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const JSONFeedVersion = "https://jsonfeed.org/version/1.1"

type JSONFeed struct {
	Version     string           `json:"version"`
	Title       string           `json:"title"`
	Description string           `json:"description,omitempty"`
	UserComment string           `json:"user_comment,omitempty"`
	Icon        string           `json:"icon,omitempty"`
	Favicon     string           `json:"favicon,omitempty"`
	Homepage    string           `json:"home_page_url,omitempty"`
	FeedURL     string           `json:"feed_url,omitempty"`
	NextURL     string           `json:"next_url,omitempty"`
	Items       []JSONFeedItem   `json:"items"`
	Authors     []JSONFeedAuthor `json:"authors,omitempty"`
	Language    string           `json:"language,omitempty"`
	Expired     bool             `json:"expired,omitempty"`
}

type JSONFeedItem struct {
	ID            string               `json:"id"`
	URL           string               `json:"url"`
	ExternalURL   string               `json:"external_url,omitempty"`
	Title         string               `json:"title,omitempty"`
	Summary       string               `json:"summary,omitempty"`
	Text          string               `json:"content_text,omitempty"`
	HTML          string               `json:"content_html,omitempty"`
	Image         string               `json:"image,omitempty"`
	BannerImage   string               `json:"banner_image,omitempty"`
	DatePublished *time.Time           `json:"date_published,omitempty"`
	DateModified  *time.Time           `json:"date_modified,omitempty"`
	Authors       []JSONFeedAuthor     `json:"authors,omitempty"`
	Tags          []string             `json:"tags,omitempty"`
	Language      string               `json:"language,omitempty"`
	Attachments   []JSONFeedAttachment `json:"attachments,omitempty"`

	MatrixEvent *event.Event `json:"_matrix_event,omitempty"`
}

type JSONFeedAuthor struct {
	Name   string `json:"name,omitempty"`
	URL    string `json:"url,omitempty"`
	Avatar string `json:"avatar,omitempty"`

	AvatarMXC id.ContentURI `json:"_avatar_mxc,omitempty"`
}

type JSONFeedAttachment struct {
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
	Title    string `json:"title,omitempty"`
	Size     int    `json:"size_in_bytes,omitempty"`
	Duration int    `json:"duration_in_seconds,omitempty"`
}
