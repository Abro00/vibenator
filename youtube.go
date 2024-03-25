package main

import (
	"github.com/kkdai/youtube/v2"
)

func FetchYoutubeUrl(url, locale string) ([]*youtube.Video, error) {
	client := youtube.Client{}

	switch {
	// case strings.Contains(url, "playlist"):
	// 	playlist, err := client.GetPlaylist(url)
	// 	if err != nil {
	// 		logger.Errorf(err.Error())
	// 		return nil, UnexpectedQueryErr{locale}
	// 	}

	// 	videos := []youtube.Video{}

	// 	for _, videoEntry := range playlist.Videos {
	// 		video, err := client.VideoFromPlaylistEntry(videoEntry)
	// 		if err != nil {
	// 			logger.Errorf(err.Error())
	// 			continue
	// 		}
	// 		videos = append(videos, *video)
	// 	}

	// 	if len(videos) == 0 {
	// 		return nil, UnexpectedQueryErr{locale}
	// 	}

	// 	return videos, nil
	default:
		video, err := client.GetVideo(url)
		if err != nil {
			logger.Errorf(err.Error())
			return nil, UnexpectedQueryErr{locale}
		}

		return []*youtube.Video{video}, nil
	}
}