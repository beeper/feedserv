package main

import (
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func (fs *FeedServ) HandleMetadata(_ mautrix.EventSource, evt *event.Event) {
	if evt.StateKey == nil || (*evt.StateKey != "" && evt.Type == event.StateMember) {
		return
	}
	feed, ok := fs.Config.feedsByRoomID[evt.RoomID]
	if !ok {
		return
	}
	feed.updateLock.Lock()
	defer feed.updateLock.Unlock()
	log := fs.Log.With().
		Str("room_id", evt.RoomID.String()).
		Str("sender", evt.Sender.String()).
		Str("event_id", evt.ID.String()).
		Str("feed_id", feed.id).
		Str("action", "feed metadata update").
		Logger()
	switch evt.Type {
	case event.StateRoomName:
		feed.title = evt.Content.AsRoomName().Name
		log.Debug().Str("feed_title", feed.title).Msg("Updated feed title")
	case event.StateTopic:
		feed.description = evt.Content.AsTopic().Topic
		log.Debug().Str("feed_description", feed.description).Msg("Updated feed description")
	case event.StateRoomAvatar:
		feed.icon = fs.Media.GetDownloadURL(evt.Content.AsRoomAvatar().URL)
		log.Debug().Str("feed_icon", feed.icon).Msg("Updated feed icon")
	case event.StatePowerLevels:
		feed.powers = evt.Content.AsPowerLevels()
		log.Debug().Msg("Updated cached power levels")
	case event.StateMember:
		userID := id.UserID(evt.GetStateKey())
		if feed.powers.GetUserLevel(userID) > feed.powers.GetEventLevel(event.EventMessage) {
			profile := evt.Content.AsMember()
			feed.authors[userID] = JSONFeedAuthor{
				Name:      profile.Displayname,
				URL:       userID.URI().MatrixToURL(),
				Avatar:    fs.Media.GetDownloadURL(profile.AvatarURL.ParseOrIgnore()),
				AvatarMXC: profile.AvatarURL.ParseOrIgnore(),
			}
			log.Debug().
				Str("user_id", userID.String()).
				Str("name", feed.authors[userID].Name).
				Str("avatar", feed.authors[userID].Avatar).
				Msg("Updated author profile")
		}
	}

	fs.regenerateFeed(feed, log)
}

func (fs *FeedServ) InitSyncFeed(feed *FeedConfig) {
	start := time.Now()
	log := fs.Log.With().
		Str("room_id", feed.RoomID.String()).
		Str("feed_id", feed.id).
		Str("action", "initial feed load").
		Logger()
	log.Debug().Msg("Syncing initial metadata")
	state, err := fs.Client.State(feed.RoomID)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch room state")
		return
	}
	feed.updateLock.Lock()
	defer feed.updateLock.Unlock()
	roomNameEvt := state[event.StateRoomName][""]
	roomTopicEvt := state[event.StateTopic][""]
	roomAvatarEvt := state[event.StateRoomAvatar][""]
	if roomNameEvt != nil {
		feed.title = roomNameEvt.Content.AsRoomName().Name
	}
	if roomTopicEvt != nil {
		feed.description = roomTopicEvt.Content.AsTopic().Topic
	}
	if roomAvatarEvt != nil {
		feed.icon = fs.Media.GetDownloadURL(roomAvatarEvt.Content.AsRoomAvatar().URL)
	}

	feed.powers = state[event.StatePowerLevels][""].Content.AsPowerLevels()
	feed.authors = make(map[id.UserID]JSONFeedAuthor)
	for userID, level := range feed.powers.Users {
		if level >= feed.powers.GetEventLevel(event.EventMessage) {
			profile := state[event.StateMember][userID.String()].Content.AsMember()
			feed.authors[userID] = JSONFeedAuthor{
				Name:      profile.Displayname,
				URL:       userID.URI().MatrixToURL(),
				Avatar:    fs.Media.GetDownloadURL(profile.AvatarURL.ParseOrIgnore()),
				AvatarMXC: profile.AvatarURL.ParseOrIgnore(),
			}
		}
	}
	resp, err := fs.Client.Messages(feed.RoomID, "", "", mautrix.DirectionBackward, &mautrix.FilterPart{Types: []event.Type{event.EventMessage}}, feed.MaxEntries)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch room messages")
	}
	for i := len(resp.Chunk) - 1; i >= 0; i-- {
		evt := resp.Chunk[i]
		_ = evt.Content.ParseRaw(evt.Type)
		feed.pushEvent(log, evt)
	}
	log.Info().
		Str("feed_title", feed.title).
		Str("feed_description", feed.description).
		Str("feed_icon", feed.icon).
		Int("entry_count", feed.entries.Size()).
		Dur("duration", time.Since(start)).
		Msg("Synced feed metadata")

	fs.regenerateFeed(feed, log)
}
