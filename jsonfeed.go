package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util"
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

func (fs *FeedServ) generateJSONFeed(feed *FeedConfig) ([]byte, string, error) {
	feedURL := fs.Config.PublicURL + feed.id + ".json"
	jsonFeed := &JSONFeed{
		Version:     JSONFeedVersion,
		Title:       feed.title,
		Description: feed.description,
		Icon:        feed.icon,
		Homepage:    feed.Homepage,
		Language:    feed.Language,
		FeedURL:     feedURL,
	}
	jsonFeed.Items, _ = util.MapRingBuffer(feed.entries, func(evtID id.EventID, evt *event.Event) (JSONFeedItem, error) {
		content := evt.Content.AsMessage()
		ts := time.UnixMilli(evt.Timestamp).UTC()
		var attachments []JSONFeedAttachment
		if content.URL != "" {
			attachments = append(attachments, JSONFeedAttachment{
				URL:      fs.Media.GetDownloadURL(content.URL.ParseOrIgnore()),
				MimeType: content.GetInfo().MimeType,
				Title:    content.FileName,
				Size:     content.GetInfo().Size,
				Duration: content.GetInfo().Duration,
			})
		}
		var editedAt *time.Time
		if !evt.Mautrix.EditedAt.IsZero() {
			editedAt = &evt.Mautrix.EditedAt
		}
		author, ok := feed.authors[evt.Sender]
		var authors []JSONFeedAuthor
		if ok {
			authors = []JSONFeedAuthor{author}
		}
		return JSONFeedItem{
			ID:   evt.ID.String(),
			URL:  evt.RoomID.EventURI(evt.ID, fs.Config.homeserverDomain).MatrixToURL(),
			Text: content.Body,
			HTML: content.FormattedBody,

			Attachments: attachments,
			Authors:     authors,

			DatePublished: &ts,
			DateModified:  editedAt,

			MatrixEvent: evt,
		}, nil
	})
	jsonData, err := json.Marshal(jsonFeed)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal JSON feed: %w", err)
	}
	return jsonData, fmt.Sprintf(`"%x"`, sha256.Sum256(jsonData)), nil
}
