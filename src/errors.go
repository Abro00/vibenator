package main

import "github.com/bwmarrin/discordgo"

type UnexpectedQueryErr struct {
	locale string
}

func (err UnexpectedQueryErr)Error() string {
	msg := "Unexpected url in query"

	switch err.locale {
	case string(discordgo.Russian):
		msg = "Неверная ссылка в запросе"
	}

	return msg
}

type UserNotInVcErr struct {
	locale string
}

func (err UserNotInVcErr)Error() string {
	msg := "To send command you must be in voice chat in this server"

	switch err.locale {
	case string(discordgo.Russian):
		msg = "Чтобы отправлять команды зайдите в голосовой канал на этом сервере"
	}

	return msg
}
