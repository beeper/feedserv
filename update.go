package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
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

	feed.pushEvent(log, evt)

	fs.regenerateFeed(feed, log)
}

func (feed *FeedConfig) pushEvent(log zerolog.Logger, evt *event.Event) {
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
}

func (fs *FeedServ) regenerateFeed(feed *FeedConfig, log zerolog.Logger) {
	log.Debug().Msg("Regenerating feed")
	start := time.Now()

	oldJSONHash := feed.jsonHash
	var err error
	feed.json, feed.jsonHash, err = fs.generateJSONFeed(feed)
	if err != nil {
		log.Err(err).Msg("Failed to generate JSON feed")
		return
	}

	gorillaFeed := fs.generateGorillaFeed(feed)
	var buf bytes.Buffer
	if err = gorillaFeed.WriteRss(&buf); err != nil {
		log.Err(err).Msg("Failed to generate RSS feed")
	} else {
		feed.rss = buf.Bytes()
		rssHash := sha256.Sum256(feed.rss)
		feed.rssHash = hex.EncodeToString(rssHash[:])
	}
	buf.Reset()
	if err = gorillaFeed.WriteAtom(&buf); err != nil {
		log.Err(err).Msg("Failed to generate Atom feed")
	} else {
		feed.atom = buf.Bytes()
		atomHash := sha256.Sum256(feed.atom)
		feed.atomHash = hex.EncodeToString(atomHash[:])
	}

	feed.lastUpdate = time.Now()
	log.Info().
		Str("old_json_hash", oldJSONHash).
		Str("new_json_hash", feed.jsonHash).
		Int("item_count", feed.entries.Size()).
		Dur("duration", time.Since(start)).
		Msg("Feed updated successfully")
}
