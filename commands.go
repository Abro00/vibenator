package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var (
	minQueryPos float64 = 1

	commands = []*discordgo.ApplicationCommand{
		{
			Name: "play",
			Description: "Request track via url",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type: discordgo.ApplicationCommandOptionString,
					Name: "query",
					// TODO: add track name and author search, spotify URL
					Description: "yt url",
					Required: true,
				},
			},
		},
		{
			Name: "stop",
			Description: "Stop player and clear query",
		},
		// {
		// 	Name: "pause",
		// 	Description: "Pause player",
		// },
		// {
		// 	Name: "resume",
		// 	Description: "Resume player",
		// },
		// {
		// 	Name: "query",
		// 	Description: "Show current player query",
		// },
		// {
		// 	Name: "leave",
		// 	Description: "Leave voice channel",
		// },
		// {
		// 	Name: "remove",
		// 	Description: "Remove track from query",
		// 	Options: []*discordgo.ApplicationCommandOption{
		// 		{
		// 			Type: discordgo.ApplicationCommandOptionInteger,
		// 			Name: "position",
		// 			Description: "track position in query",
		// 			Required: true,
		// 			MinValue: &minQueryPos,
		// 		},
		// 	},
		// },
		// {
		// 	// TODO: сделать опции через iota-const, определенные в плеере
		// 	Name: "loop",
		// 	Description: "Loop the player",
		// 	Options: []*discordgo.ApplicationCommandOption{
		// 		{
		// 			Name: "kind",
		// 			Description: "Loop track / queue / none",
		// 			Type: discordgo.ApplicationCommandOptionInteger,
		// 			Choices: []*discordgo.ApplicationCommandOptionChoice{
		// 				{
		// 					Name: "track",
		// 					Value: 1,
		// 				},
		// 				{
		// 					Name: "queue",
		// 					Value: 2,
		// 				},
		// 				{
		// 					Name: "none",
		// 					Value: 0,
		// 				},
		// 			},
		// 		},
		// 	},
		// },
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, vcID string){
		"play":   playCmdHandler,
		"stop":   stopCmdHandler,
		// "pause":  pauseCmdHandler,
		// "resume": resumeCmdHandler,
		// "query":  queryCmdHandler,
		// "leave":  leaveCmdHandler,
		// "remove": removeCmdHandler,
		// "loop":   loopCmdHandler,
	}
)

// register commands globally
func registerAllCommands(s *discordgo.Session) {
	logger.Infof("Start commands registration")
	for _, cmd := range commands {
		logger.Infof("  try register [%s] command. . .\r", cmd.Name)

		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			logger.Errorf("  failed register command [%s]: %s", cmd.Name, err.Error())
			continue
		}
		logger.Infof("  [%s] registered successfully!", cmd.Name)
	}
}

func addCommandsHandler(s *discordgo.Session){
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		vcID, err := getUserVcID(s, i)
		if err != nil {
			ErrorInteractionResponse(s, i, err.Error())
			logger.Errorf(err.Error())
			return
		}
		if vcID == "" {
			ErrorInteractionResponse(s, i, UserNotInVcErr{string(i.Locale)}.Error())
			return
		}

		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i, vcID)
		}
	})
}

// TODO: add flag to run program just for clearing bot's interactions
func unregisterAllCommands(s *discordgo.Session) {
	globalCommands, err := s.ApplicationCommands(s.State.User.ID, "")
	if cfg.Debug { logger.Debugf("=== clearing global commands ===") }
	for _, cmd := range globalCommands {
		if cfg.Debug { logger.Debugf("%+v", *cmd) }
		err = s.ApplicationCommandDelete(s.State.User.ID, cmd.GuildID, cmd.ID)
		if err != nil {
			logger.Errorf(err.Error())
		}
	}

	for _, guild := range s.State.Guilds {
		guildCommands, err := s.ApplicationCommands(s.State.User.ID, guild.ID)
		if cfg.Debug { logger.Debugf("=== clearing guild commands ===") }
		for _, cmd := range guildCommands {
			if cfg.Debug { logger.Debugf("%+v", *cmd) }
			err = s.ApplicationCommandDelete(s.State.User.ID, cmd.GuildID, cmd.ID)
			if err != nil {
				logger.Errorf(err.Error())
			}
		}
	}
}

func playCmdHandler(s *discordgo.Session, i *discordgo.InteractionCreate, vcID string) {
	options := i.ApplicationCommandData().Options
	optionsMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionsMap[opt.Name] = opt
	}

	query, ok := optionsMap["query"]
	if !ok {
		// TODO: respond to discord
		logger.Errorf("Wrong [%s] command options: %+v", i.ApplicationCommandData().Name, optionsMap)
		return
	}

	if cfg.Debug {logger.Debugf("Sent /play command with query: %s", query.StringValue())}

	videos, err := FetchTracksData(query.StringValue(), string(i.Locale))
	if err != nil {
		ErrorInteractionResponse(s, i, err.Error())
		logger.Errorf("error: %s; query: %s", err.Error(), query.StringValue())
		return
	}

	// TODO: вынести в функцию?
	// (re)connect to voice channel
	vc, ok := s.VoiceConnections[i.GuildID]
	if !ok {
		vc, err = s.ChannelVoiceJoin(i.GuildID, vcID, false, true)
		if err != nil {
			ErrorInteractionResponse(s, i, err.Error())
			logger.Errorf(err.Error())
			return
		}
	} else if vcID != vc.ChannelID {
		ErrorInteractionResponse(s, i, "TODO: suggest user to move bot in this channel")
		return
	}

	player := PlayersMap[i.GuildID]
	if player == nil {
		player = NewPlayer(vc)
		player.Lock()
		PlayersMap[i.GuildID] = player
		player.Unlock()
	}
	player.Add(videos)
	// TODO: Format response to user
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Added `%s`", TrackString(videos[0])),
		},
	})

	player.Play()
}

func stopCmdHandler(s *discordgo.Session, i *discordgo.InteractionCreate, vcID string) {
	if cfg.Debug { logger.Debugf("Sent /stop command") }

	player, ok := PlayersMap[i.GuildID]
	if ok {
		player.Clear()
		player.Stop()
		delete(PlayersMap, i.GuildID)
	}
	// TODO: Format response to user
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Player stopped"),
		},
	})
}
