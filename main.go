package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	"golang.org/x/net/context"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util"
)

func makeClient(cfg *Config, log *zerolog.Logger) (*mautrix.Client, error) {
	cli, err := mautrix.NewClient(cfg.HomeserverURL, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}
	cli.Log = log.With().Str("component", "matrix").Logger()
	_, err = cli.Login(&mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: cfg.UserID.String(),
		},
		Password:                 cfg.Password,
		DeviceID:                 "feedserv",
		InitialDeviceDisplayName: "feedserv",
		StoreCredentials:         true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to log in: %w", err)
	}
	cfg.homeserverDomain = cli.UserID.Homeserver()
	return cli, nil
}

type FeedServ struct {
	Config *Config
	Client *mautrix.Client
	Media  *mautrix.Client
	Log    *zerolog.Logger
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log, err := cfg.LogConfig.Compile()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to compile log config:", err)
		os.Exit(1)
	}
	cli, err := makeClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize mautrix client")
	}
	mediaCli, err := mautrix.NewClient(cfg.MediaURL, "", "")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize media client")
	}

	fs := &FeedServ{
		Config: cfg,
		Client: cli,
		Media:  mediaCli,
		Log:    log,
	}

	var wg sync.WaitGroup
	cfg.feedsByRoomID = make(map[id.RoomID]*FeedConfig)
	wg.Add(len(cfg.Feeds))
	allowedRoomIDs := make([]id.RoomID, 0, len(cfg.Feeds))
	for feedID, feed := range cfg.Feeds {
		if feed.RoomID == "" && feed.RoomAlias != "" {
			resp, err := fs.Client.ResolveAlias(feed.RoomAlias)
			if err != nil {
				log.Fatal().
					Str("room_alias", feed.RoomAlias.String()).
					Str("feed_id", feedID).
					Msg("Failed to resolve room ID for feed")
			}
			feed.RoomID = resp.RoomID
			log.Debug().
				Str("room_alias", feed.RoomAlias.String()).
				Str("room_id", feed.RoomID.String()).
				Str("feed_id", feedID).
				Msg("Resolved room ID for feed")
		}
		feed.id = feedID
		feed.entries = util.NewRingBuffer[id.EventID, *event.Event](feed.MaxEntries)
		if existing, alreadyExists := cfg.feedsByRoomID[feed.RoomID]; alreadyExists {
			log.Fatal().
				Str("room_id", feed.RoomID.String()).
				Str("prev_feed_id", existing.id).
				Str("new_feed_id", feedID).
				Msg("Multiple feeds pointing at same room")
		}
		go func() {
			fs.InitSyncFeed(feed)
			wg.Done()
		}()
		cfg.feedsByRoomID[feed.RoomID] = feed
		allowedRoomIDs = append(allowedRoomIDs, feed.RoomID)
	}
	wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(2)

	syncer := cli.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, fs.HandleFeedEvent)
	syncer.OnEventType(event.StateMember, fs.HandleInvite)
	syncer.OnEventType(event.StateMember, fs.HandleMetadata)
	syncer.OnEventType(event.StatePowerLevels, fs.HandleMetadata)
	syncer.OnEventType(event.StateRoomName, fs.HandleMetadata)
	syncer.OnEventType(event.StateTopic, fs.HandleMetadata)
	syncer.OnEventType(event.StateRoomAvatar, fs.HandleMetadata)

	nothing := mautrix.FilterPart{NotTypes: []event.Type{{Type: "*"}}}
	importantTypes := mautrix.FilterPart{
		Types: []event.Type{
			event.EventMessage, event.StateMember, event.StatePowerLevels,
			event.StateRoomName, event.StateTopic, event.StateRoomAvatar,
		},
	}
	syncer.FilterJSON = &mautrix.Filter{
		AccountData: nothing,
		Presence:    nothing,
		Room: mautrix.RoomFilter{
			AccountData: nothing,
			Ephemeral:   nothing,
			Rooms:       allowedRoomIDs,
			State:       importantTypes,
			Timeline:    importantTypes,
		},
	}

	cli.Store = mautrix.NewAccountDataStore("com.beeper.feedserv_sync_token", cli)

	server := http.Server{
		Addr:    cfg.ListenAddress,
		Handler: fs,
	}
	go func() {
		defer wg.Done()
		err := cli.SyncWithContext(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Fatal().Err(err).Msg("Error in syncer")
		} else {
			log.Debug().Msg("Syncer finished cleanly")
		}
	}()
	go func() {
		defer wg.Done()
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Error in HTTP server")
		} else {
			log.Debug().Msg("HTTP server finished cleanly")
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Info().Msg("Interrupt received, stopping...")

	cancel()
	err = server.Close()
	if err != nil {
		log.Warn().Err(err).Msg("Error closing HTTP server")
	}
	wg.Wait()
	log.Info().Msg("Feedserv stopped")
}
