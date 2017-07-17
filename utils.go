package main

import (
	"io/ioutil"
	"os"

	"github.com/microwaves/go-utils/logger"
)

func setupLogging() {
	f, _ := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	logger.NewLogger(ioutil.Discard, f, f, f)
}

func handleError(e error) {
	if e != nil {
		logger.Error.Println(e)
	}
}
