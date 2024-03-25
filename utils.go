package main

import (
	"fmt"
	"os"
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
	return fmt.Sprintf("%s | %s %s", video.Author, video.Title, FormatDuration(video.Duration))
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