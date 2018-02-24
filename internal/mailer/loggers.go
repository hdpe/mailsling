package mailer

import (
	"io/ioutil"
	"log"
)

type Loggers struct {
	Info  *log.Logger
	Error *log.Logger
}

var NOOPLog = &Loggers{
	Info:  log.New(ioutil.Discard, "", 0),
	Error: log.New(ioutil.Discard, "", 0),
}
