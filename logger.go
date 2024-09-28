package gpool

import "fmt"

type Logger interface {
	Info(msg string)
	Error(err error)
}

type logger struct{}

func (*logger) Info(msg string) {
	fmt.Println(msg)
}

func (*logger) Error(err error) {
	fmt.Println(err.Error())
}

func newLogger() Logger {
	return &logger{}
}
