package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	JSONFeedMime = "application/feed+json"
	RSSMime      = "application/rss+xml"
)

func writeError(w http.ResponseWriter, status int, error string, args ...any) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": fmt.Sprintf(error, args...)})
}

func (fs *FeedServ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	feedPath := strings.ToLower(r.URL.Path)
	ext := path.Ext(feedPath)
	feedPath = feedPath[:len(feedPath)-len(ext)]
	var mime string
	switch ext {
	case ".json", "":
		mime = JSONFeedMime
	case ".rss":
		mime = RSSMime
	default:
		writeError(w, http.StatusNotFound, "Unsupported feed type %q", ext)
		return
	}

	feed, ok := fs.Config.Feeds[feedPath]
	if !ok {
		writeError(w, http.StatusNotFound, "Feed %q not found", feedPath)
		return
	}

	feed.updateLock.RLock()
	var data []byte
	var hash string
	lastMod := feed.lastUpdate
	switch mime {
	case JSONFeedMime:
		data = feed.json
		hash = feed.jsonHash
	case RSSMime:
		data = feed.rss
		hash = feed.rssHash
	default:
		panic(fmt.Errorf("incorrect mime %q", mime))
	}
	feed.updateLock.RUnlock()

	w.Header().Add("Last-Modified", lastMod.UTC().Format(http.TimeFormat))
	w.Header().Add("ETag", hash)

	if r.Header.Get("If-None-Match") == hash {
		w.WriteHeader(http.StatusNotModified)
		return
	} else if ifModifiedSinceStr := r.Header.Get("If-Modified-Since"); ifModifiedSinceStr != "" {
		ifModifiedSince, err := time.Parse(http.TimeFormat, ifModifiedSinceStr)
		if err == nil && !ifModifiedSince.After(lastMod) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	w.Header().Add("Content-Type", mime)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
