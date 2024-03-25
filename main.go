package main

import (
	"context"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	cfg Config
	logger Logger
)

type Config struct {
	BotToken string
	Debug 	 bool
}

func init() {
	cfg = Config{
		BotToken: getEnv("DC_BOT_TOKEN", ""),
		Debug: 		getEnv("DEBUG", "") != "",
	}
	logger = NewLogger()
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	if cfg.BotToken == "" {
		logger.Fatalf("Empty token")
		return
	}

	// create new bot session
	logger.Infof("Trying to register bot . . .")
	s, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		logger.Fatalf(err.Error())
		return
	}
	logger.Infof("Connected succesfully")

	wg := &sync.WaitGroup{}
	wg.Add(1)
	// ready event handler
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		logger.Infof("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		err = s.UpdateListeningStatus("^..^")
		if err != nil {
			logger.Errorf(err.Error())
		}

		wg.Done()
	})

	// open bot session
	err = s.Open()
	if err != nil {
		logger.Fatalf("Cannot open the session: %v", err)
		return
	}
	defer s.Close()

	wg.Wait()

	registerAllCommands(s)
	defer unregisterAllCommands(s)

	addCommandsHandler(s)

	// block main goroutine, waiting to stop process
	logger.Infof("Running now. Press Ctrl+C to stop process")

	<-ctx.Done()
	logger.Infof("Gracefully shutdown")
}
