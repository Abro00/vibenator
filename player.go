package main

import (
	"container/list"
	"context"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca/v2"
	"github.com/kkdai/youtube/v2"
)

var (
	// mapping GuildID with player instance
	PlayersMap = make(map[string]*Player)
)

var PlayerStates = [3]uint8{PlayerStoppedState, PlayerPlayingState, PlayerPausedState}
const (
	PlayerStoppedState uint8 = iota
	PlayerPlayingState
	PlayerPausedState
	PlayerAfkTimeout = time.Minute
)

type PlayerCtx struct {
	context.Context
	cancel context.CancelFunc
}

type Player struct {
	sync.RWMutex

	// When initialised player is stopped
	// Stopped - 0 | Playing - 1 | Paused - 2
	state uint8
	StateChanged chan bool

	// Current playing track
	CurrentPlaying *youtube.Video

	vc *discordgo.VoiceConnection
	PlayerQueue *list.List

	cbMap map[uint8][]func()
	timer *time.Timer

	ctx PlayerCtx
}

func NewPlayer(vc *discordgo.VoiceConnection) *Player {
	// TODO: запускать таймер когда закончилась очередь и плеер в состоянии stopped -> сбасывать как только запускается новый трек
	// ИЛИ если голосовом канале не осталось пользователей кроме бота -> сбасывать как только пользователь заходит в чат VoiceStateUpdate event
	// time.NewTimer()
	ctx, cancel := context.WithCancel(context.Background())
	plr := &Player{
		PlayerQueue:  list.New(),
		timer:        time.NewTimer(PlayerAfkTimeout),
		ctx:          PlayerCtx{ctx, cancel},
		StateChanged: make(chan bool),
		vc:           vc,
	}

	// start streaming goroutine
	go plr.stream()

	return plr
}

func (plr *Player)GetState() uint8 {
	plr.RLock()
	state := plr.state
	plr.RUnlock()
	return state
}

func (plr *Player)AddCallback(state uint8, callback func()) {
	plr.Lock()
	plr.cbMap[state] = append(plr.cbMap[state], callback)
	plr.Unlock()
}

func (plr *Player)ClearCallbacks(state uint8) {
	plr.Lock()
	plr.cbMap[state] = []func(){}
	plr.Unlock()
}

func (plr *Player)changeState(state uint8) {
	plr.Lock()
	plr.state = state
	callbacks := plr.cbMap[state]
	close(plr.StateChanged)
	plr.StateChanged = make(chan bool)
	plr.Unlock()
	// run callbacks when enter new state
	for _, cb := range callbacks {
		cb()
	}
}

func (plr *Player)Add(videos []*youtube.Video) {
	plr.Lock()
	for _, video := range videos {
		plr.PlayerQueue.PushBack(video)
	}
	plr.Unlock()
}

func (plr *Player)Play() {
	if plr.GetState() != PlayerStoppedState {
		return
	}

	if cfg.Debug { logger.Debugf("playing music. queue: %d", plr.PlayerQueue.Len()) }
	plr.changeState(PlayerPlayingState)
}

func (plr *Player)Stop() {
	plr.changeState(PlayerStoppedState)
	plr.ctx.cancel()
}

func (plr *Player)Clear() {
	plr.Lock()
	plr.PlayerQueue.Init()
	plr.Unlock()
}

// always must .Cleanup() session after use
func getEncodingSession(video *youtube.Video) (*dca.EncodeSession, error) {
	streamUrl, err := getStreamUrl(video)
	if err != nil {
		return nil, err
	}

	options := dca.StdEncodeOptions
	// options.Application = dca.AudioApplicationLowDelay
	options.Channels = 1
	// options.Bitrate = 96
	// options.Bitrate = 128
	// options.FrameRate = 96000

	encodingSession, err := dca.EncodeFile(streamUrl, options)
	if err != nil {
		return nil, err
	}

	return encodingSession, nil
}

func getStreamUrl(video *youtube.Video) (string, error) {

	audioFormats := video.Formats.Type("audio/webm")
	sort.SliceStable(audioFormats, video.SortBitrateDesc)

	client := youtube.Client{}
	streamUrl, err := client.GetStreamURL(video, &(audioFormats[0]))
	if err != nil {
		return "", err
	}

	return streamUrl, nil
}

// run go stream to create streaming session
// stream works in two loops:
// first - getting track from queue, if queue is empty, cancel context and change state to stopped
// second - sending opus to vc and breaks when opus ended; if
func (plr *Player)stream() {
	var encodingSession *dca.EncodeSession
	defer func() {
		plr.vc.Disconnect()
		plr.ctx.cancel()
		if encodingSession != nil { encodingSession.Cleanup() }
		delete(PlayersMap, plr.vc.GuildID)
	}()

	opusChan := make(chan []byte)
	// receive frames in loop
	go func() {
		for {
			breakLoop := func() bool {
				select {
				case opus := <- opusChan:
					plr.vc.OpusSend <- opus
					return false
				case <- plr.ctx.Done():
					return true
				}
			}()
			if breakLoop { break }
		}
	}()


	// load new track
	for {
		breakLoop := func() bool {
			switch plr.GetState() {
			case PlayerPlayingState:
				// get first video from list
				// if list is empty change state to stopped and run timer
				plr.Lock()
				plr.timer.Stop()
				track := plr.CurrentPlaying
				plr.Unlock()

				if track == nil {
					plr.Lock()
					if plr.PlayerQueue.Len() < 1 {
						plr.Unlock()
						logger.Infof("Queue ended")
						plr.changeState(PlayerStoppedState)
						return false
					}

					elem := plr.PlayerQueue.Front()
					queueVal := plr.PlayerQueue.Remove(elem)
					queueTrack, ok := queueVal.(*youtube.Video)
					if !ok {
						logger.Errorf("Error get track from queue: %#v", queueVal)
						plr.Unlock()
						plr.changeState(PlayerStoppedState)
						return false
					}

					track = queueTrack
					plr.CurrentPlaying = track
					plr.Unlock()
				}

				if encodingSession == nil {
					es, err := getEncodingSession(track)
					if err != nil {
						logger.Errorf("Error getting encoding session: %s", err.Error())
						plr.changeState(PlayerStoppedState)
						encodingSession = nil
						return false
					}
					encodingSession = es
				}

				for {
					opus, err := encodingSession.OpusFrame()
					if err != nil {
						if err != io.EOF {
							logger.Errorf("Error reading opus: %s", err.Error())
							plr.changeState(PlayerStoppedState)
						}
						// REMOVE
						// TODO: плеер выкидывает ошибку на последних байтах, не доигрывая трек до конца
						logger.Debugf("EOF err: %#v, opus(%d): %#v ", err, len(opus), opus)
						plr.Lock()
						plr.CurrentPlaying = nil
						plr.Unlock()
						encodingSession.Cleanup()
						encodingSession = nil
						return false
					}

					select {
					case <- plr.StateChanged:
						return false
					case <- plr.ctx.Done():
						return true
					case opusChan <- opus:
						// just go read next frame
					}
				}
			case PlayerStoppedState:
				if encodingSession != nil {
					encodingSession.Cleanup()
					encodingSession = nil
				}
				plr.Lock()
				plr.timer.Stop()
				plr.timer = time.NewTimer(PlayerAfkTimeout)
				plr.CurrentPlaying = nil
				plr.Unlock()

				select {
				case <- plr.StateChanged:
					return false
				case <- plr.timer.C:
					plr.ctx.cancel()
					return true
				case <- plr.ctx.Done():
					return true
				}
			case PlayerPausedState:
				plr.Lock()
				plr.timer.Stop()
				plr.Unlock()

				select {
				case <- plr.StateChanged:
					return false
				case <- plr.ctx.Done():
					return true
				}
			default:
				// unknown player state; close player
				return true
			}
		}()

		if breakLoop {
			break
		}
	}
}