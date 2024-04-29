package main

import (
	"container/list"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
)

const (
  interactionErrorFmt = "Oops smth went wrong. Error report:\n```\n%s\n```"
)

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
    return val
  }
  return fallback
}

func TrackString(video *youtube.Video) string {
	return fmt.Sprintf("[%s] %s - %s", video.Author, video.Title, FormatDuration(video.Duration))
}

func QueueString(queue list.List) string {
  result := ""
  elem := queue.Front()
  if elem != nil {
    for i := 0; i < 10; i++ {
      queueVal := elem.Value
      queueTrack := queueVal.(*youtube.Video)
      result += fmt.Sprintf("%d. %s\n", i+1, TrackString(queueTrack))

      elem = elem.Next()
      if elem == nil { return result }
    }
  }
  return result
}

func FormatDuration(d time.Duration) string {
  d = d.Round(time.Second)
  hr := d / time.Hour
  d -= hr * time.Hour
  min := d / time.Minute
  d -= min * time.Minute
  sec := d / time.Second

  res := fmt.Sprintf("%02d:%02d", min, sec)
  if hr > 0 { res = fmt.Sprintf("%02d:%s", hr, res) }

  return res
}

func ErrorInteractionResponse(s *discordgo.Session, i *discordgo.InteractionCreate, err string) {
  s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
    Type: discordgo.InteractionResponseChannelMessageWithSource,
    Data: &discordgo.InteractionResponseData{
      Content: fmt.Sprintf(interactionErrorFmt, err),
    },
  })
}

func getUserVcID(s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	g, err := s.State.Guild(i.GuildID)
	if err != nil {
		return "", err
	}
	vcID := ""
	for _, v := range g.VoiceStates {
		if v.UserID == i.Member.User.ID {
			vcID = v.ChannelID
			break
		}
	}
  return vcID, nil
}

func PrintStats() {
  var m runtime.MemStats
  runtime.ReadMemStats(&m)

  logger.Debugf("Alloc: %v MiB | Sys: %v MiB | Goroutines: %v", bToMb(m.Alloc), bToMb(m.Sys), runtime.NumGoroutine())
}

func bToMb(b uint64) uint64 {
  return b / 1024 / 1024
}
