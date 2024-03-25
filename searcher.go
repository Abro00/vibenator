package main

import (
	"regexp"

	"github.com/kkdai/youtube/v2"
)

const (
	ytUrlRegex = `^(?:https?\:\/\/)?(?:www\.)?(?:(?:youtube\.com\/)|(?:youtu.be\/))`
)

const (
	_ = iota
	ytSrv
	spotifySrv
)

func FetchTracksData(query, locale string) ([]*youtube.Video, error) {
	switch extractService(query) {
	case ytSrv:
		return FetchYoutubeUrl(query, locale)
	}

	return []*youtube.Video{}, UnexpectedQueryErr{locale}
}

func extractService(url string) uint8 {
	rgx := regexp.MustCompile(ytUrlRegex)
	if rgx.MatchString(url) {
		return ytSrv
	}

	// rgx = regexp.MustCompile(spotifyUrlRegex)
	// if rgx.MatchString(url) {
	// 	return spotifySrv
	// }

	return 0
}
