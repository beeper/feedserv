package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"go.mau.fi/zeroconfig"
	"gopkg.in/yaml.v3"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util"
)

type Config struct {
	HomeserverURL string    `yaml:"homeserver_url"`
	MediaURL      string    `yaml:"media_url"`
	UserID        id.UserID `yaml:"user_id"`
	Password      string    `yaml:"password"`

	LogConfig zeroconfig.Config `yaml:"logging"`

	ListenAddress string `yaml:"listen_address"`
	PublicURL     string `yaml:"public_url"`

	CloudflareZoneID string `yaml:"cloudflare_zone_id"`
	CloudflareToken  string `yaml:"cloudflare_token"`

	Feeds         map[string]*FeedConfig `yaml:"feeds"`
	feedsByRoomID map[id.RoomID]*FeedConfig

	homeserverDomain string
}

type FeedConfig struct {
	RoomAlias  id.RoomAlias `yaml:"room_alias"`
	RoomID     id.RoomID    `yaml:"room_id"`
	MaxEntries int          `yaml:"max_entries"`
	Homepage   string       `yaml:"homepage"`
	Language   string       `yaml:"language"`

	id          string
	title       string
	description string
	icon        string
	iconMXC     id.ContentURI
	authors     map[id.UserID]JSONFeedAuthor
	powers      *event.PowerLevelsEventContent

	entries    *util.RingBuffer[id.EventID, *event.Event]
	lastUpdate time.Time
	updateLock sync.RWMutex

	rss      []byte
	rssHash  string
	atom     []byte
	atomHash string
	json     []byte
	jsonHash string
}

func loadConfig() (*Config, error) {
	cfgPath := os.Getenv("FEEDSERV_CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}
	file, err := os.Open(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	var config Config
	err = yaml.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if passwordEnv := os.Getenv("FEEDSERV_PASSWORD"); passwordEnv != "" {
		config.Password = passwordEnv
	} else if passwordFileEnv := os.Getenv("FEEDSERV_PASSWORD_FILE"); passwordFileEnv != "" {
		pwBytes, err := os.ReadFile(passwordFileEnv)
		if err != nil {
			return nil, fmt.Errorf("failed to read password from file: %w", err)
		}
		config.Password = string(pwBytes)
	}
	return &config, nil
}
