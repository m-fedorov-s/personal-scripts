package main

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

func GetCommandMatchFunction(cmd, botHandle string) func(*models.Update) bool {
	return func(update *models.Update) bool {
		if update.Message == nil {
			return false
		}
		if update.Message.Chat.Type == "private" {
			return update.Message.Text == cmd
		} else {
			return update.Message.Text == fmt.Sprintf("%v@%v", cmd, botHandle)
		}
	}
}
