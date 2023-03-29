package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util"
)

func (fs *FeedServ) HandleFeedEvent(_ mautrix.EventSource, evt *event.Event) {
	log := fs.Log.With().
		Str("event_id", evt.ID.String()).
		Str("sender", evt.Sender.String()).
		Str("room_id", evt.RoomID.String()).
		Str("action", "new message").
		Logger()
	feed, ok := fs.Config.feedsByRoomID[evt.RoomID]
	if !ok {
		log.Debug().Msg("Dropping event in feed without room")
		return
	}
	log = log.With().Str("feed_id", feed.id).Logger()
	log.Debug().Msg("Received new event in feed room")

	feed.updateLock.Lock()
	defer feed.updateLock.Unlock()

	content := evt.Content.AsMessage()
	if edits := content.RelatesTo.GetReplaceID(); edits != "" {
		log = log.With().Str("edit_target_event_id", edits.String()).Logger()
		existingEvt, found := feed.entries.Get(edits)
		if !found {
			log.Warn().Msg("Couldn't find edit target event")
			return
		} else if existingEvt.Sender != evt.Sender {
			log.Warn().
				Str("orig_sender", existingEvt.Sender.String()).
				Msg("Dropping edit of message by different sender")
			return
		} else {
			log.Info().
				Str("original_event_id", existingEvt.ID.String()).
				Msg("Overriding content of original event with edit")
			existingEvt.Content.Parsed = content.NewContent
			existingEvt.Type = evt.Type
			existingEvt.Mautrix.EditedAt = time.UnixMilli(evt.Timestamp).UTC()
		}
	} else {
		feed.entries.Push(evt.ID, evt)
	}

	fs.regenerateFeed(feed, log)
}

func (fs *FeedServ) regenerateFeed(feed *FeedConfig, log zerolog.Logger) {
	log.Debug().Msg("Regenerating feed")
	start := time.Now()
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
	var err error
	feed.json, err = json.Marshal(jsonFeed)
	if err != nil {
		log.Err(err).Msg("Failed to marshal JSON feed")
		return
	}
	feedHash := sha256.Sum256(feed.json)
	oldJSONHash := feed.jsonHash
	feed.jsonHash = hex.EncodeToString(feedHash[:])
	feed.lastUpdate = time.Now()
	log.Info().
		Str("old_json_hash", oldJSONHash).
		Str("new_json_hash", feed.jsonHash).
		Int("item_count", len(jsonFeed.Items)).
		Dur("duration", time.Since(start)).
		Msg("Feed updated successfully")
}
