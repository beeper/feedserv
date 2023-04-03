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
	AtomMime     = "application/atom+xml"
)

func writeError(w http.ResponseWriter, status int, error string, args ...any) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": fmt.Sprintf(error, args...)})
}

func (fs *FeedServ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	feedPath := strings.ToLower(r.URL.Path)
	log := fs.Log.With().
		Str("feed_path", feedPath).
		Str("method", r.Method).
		Str("cloudflare_remote_ip", r.Header.Get("CF-Connecting-IP")).
		Logger()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		log.Debug().Msg("Requested with incorrect HTTP method")
		writeError(w, http.StatusMethodNotAllowed, "Unsupported method %q", r.Method)
		return
	}
	ext := path.Ext(feedPath)
	feedPath = feedPath[:len(feedPath)-len(ext)]
	var mime string
	switch ext {
	case ".json", "":
		mime = JSONFeedMime
	case ".rss":
		mime = RSSMime
	case ".atom":
		mime = AtomMime
	default:
		log.Debug().Msg("Requested unsupported feed type")
		writeError(w, http.StatusNotFound, "Unsupported feed type %q", ext)
		return
	}

	feed, ok := fs.Config.Feeds[feedPath]
	if !ok {
		log.Debug().Msg("Requested unknown feed")
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
	case AtomMime:
		data = feed.atom
		hash = feed.atomHash
	default:
		panic(fmt.Errorf("incorrect mime %q", mime))
	}
	feed.updateLock.RUnlock()

	w.Header().Add("Last-Modified", lastMod.UTC().Format(http.TimeFormat))
	w.Header().Add("ETag", hash)
	w.Header().Add("Cache-Control", "public, max-age=60, s-maxage=60, stale-while-revalidate=60, stale-if-error=86400")

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
	if r.Method != http.MethodHead {
		_, _ = w.Write(data)
	}
	log.Debug().
		Str("hash", hash).
		Dur("duration", time.Since(start)).
		Msg("Served feed")
}
