package main

import (
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

func (fs *FeedServ) HandleInvite(_ mautrix.EventSource, evt *event.Event) {
	if evt.GetStateKey() != fs.Client.UserID.String() || evt.Content.AsMember().Membership != event.MembershipInvite {
		return
	}
	log := fs.Log.With().
		Str("room_id", evt.RoomID.String()).
		Str("sender", evt.Sender.String()).
		Str("event_id", evt.ID.String()).
		Str("action", "invite").
		Logger()
	_, allowed := fs.Config.feedsByRoomID[evt.RoomID]
	if !allowed {
		log.Info().Msg("Rejecting invite to non-feed room")
		_, err := fs.Client.LeaveRoom(evt.RoomID)
		if err != nil {
			log.Err(err).Msg("Failed to reject invite")
		} else {
			log.Debug().Msg("Rejected invite")
		}
	} else {
		log.Info().Msg("Accepting invite to feed room")
		_, err := fs.Client.JoinRoomByID(evt.RoomID)
		if err != nil {
			log.Err(err).Msg("Failed to accept invite")
		} else {
			log.Debug().Msg("Accepted invite")
		}
	}
}
