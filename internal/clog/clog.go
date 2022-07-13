package clog

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

type Log struct {
	Label   string
	OccTime time.Time
	Err     error
}

func (l *Log) Error() string {
	return fmt.Sprintf("%v (%v) --> %v", l.Label, l.OccTime.Format(`15:04:05`), l.Err)
}

func (l *Log) Unwrap() error {
	return l.Err
}

func ToLog(label string, err error) error {
	return &Log{
		Label:   label,
		OccTime: time.Now(),
		Err:     err,
	}
}

func FuncName() string {
	counter, _, _, success := runtime.Caller(1)
	if !success {
		return "—"
	}
	s := runtime.FuncForPC(counter).Name()
	n := strings.LastIndex(s, "/")
	if n >= 0 {
		return s[n+1:]
	}
	return s
}
