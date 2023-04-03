package main

import (
	"time"

	"github.com/gorilla/feeds"
	"golang.org/x/net/html"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util"
)

func (fs *FeedServ) generateGorillaFeed(feed *FeedConfig) *feeds.Feed {
	feedURL := fs.Config.PublicURL + feed.id + ".json"
	items, _ := util.MapRingBuffer(feed.entries, func(evtID id.EventID, evt *event.Event) (*feeds.Item, error) {
		content := evt.Content.AsMessage()
		var attachment *feeds.Enclosure
		if content.URL != "" {
			attachment = &feeds.Enclosure{
				Url:  fs.Media.GetDownloadURL(content.URL.ParseOrIgnore()),
				Type: content.GetInfo().MimeType,
			}
		}
		author := feed.authors[evt.Sender]
		contentText := content.FormattedBody
		if contentText == "" {
			contentText = html.EscapeString(content.Body)
		}
		eventLink := evt.RoomID.EventURI(evt.ID, fs.Config.homeserverDomain).MatrixToURL()
		return &feeds.Item{
			Author:      &feeds.Author{Name: author.Name},
			Link:        &feeds.Link{Href: eventLink},
			Id:          eventLink,
			Updated:     evt.Mautrix.EditedAt,
			Created:     time.UnixMilli(evt.Timestamp).UTC(),
			Title:       feed.title,
			Content:     contentText,
			Description: contentText,
			Enclosure:   attachment,
		}, nil
	})
	return &feeds.Feed{
		Title:       feed.title,
		Description: feed.description,
		Link:        &feeds.Link{Href: feedURL},
		Items:       items,
		Image:       &feeds.Image{Url: feed.icon, Link: feed.icon, Title: feed.title},
		Updated:     feed.lastUpdate,
	}
}
