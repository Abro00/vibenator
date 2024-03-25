package main


import (
	"fmt"
	"log"
)

type Logger struct {}

func NewLogger() Logger {
	var l Logger
	return l
}

func (logger Logger) Infof(str string, args ...any) {
	msg := fmt.Sprintf(str, args...)
	log.Println("[INFO]: ", msg)
}

func (logger Logger) Debugf(str string, args ...any) {
	msg := fmt.Sprintf(str, args...)
	log.Println("[DEBUG]: ", msg)
}

func (logger Logger) Errorf(str string, args ...any) {
	msg := fmt.Sprintf(str, args...)
	log.Println("[ERROR]: ", msg)
}

func (logger Logger) Fatalf(str string, args ...any) {
	msg := fmt.Sprintf(str, args...)
	log.Println("[FATAL]: ", msg)
}
