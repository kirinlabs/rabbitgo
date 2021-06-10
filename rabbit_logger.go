package rabbitgo

import (
	"fmt"
	"io"
	"log"
)

const (
	LOG_DEBUG int = iota
	LOG_INFO
	LOG_WARN
	LOG_ERROR
)

const (
	LOG_PREFIX = "[rabbitgo]"
	LOG_FORMAT = log.LstdFlags
	LOG_LEVEL  = LOG_DEBUG
)

type Logger struct {
	level  int
	format int
	LogI   *log.Logger
	LogW   *log.Logger
	LogD   *log.Logger
	LogE   *log.Logger
	prefix string
	output io.Writer
}

func newLogger(o io.Writer) *Logger {
	logger := &Logger{output: o, level: LOG_LEVEL, prefix: LOG_PREFIX, format: LOG_FORMAT}
	logger.Init()
	return logger
}

func (l *Logger) Init() {
	l.LogD = log.New(l.output, l.color(32, "DEBUG"), l.format)
	l.LogI = log.New(l.output, l.color(32, "INFO"), l.format)
	l.LogW = log.New(l.output, l.color(33, "WARN"), l.format)
	l.LogE = log.New(l.output, l.color(31, "ERROR"), l.format)
	return
}

func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
	l.Init()
	return
}

func (l *Logger) Prefix() string {
	return l.prefix
}

func (l *Logger) Flag() int {
	return l.format
}

func (l *Logger) SetFlag(flag int) {
	l.format = flag
	l.Init()
	return
}

func (l *Logger) Level() int {
	return l.level
}

func (l *Logger) SetLevel(level int) {
	l.level = level
	return
}

func (l *Logger) Debug(v ...interface{}) {
	if l.level <= LOG_DEBUG {
		l.LogD.Output(2, fmt.Sprint(v...))
	}
	return
}

func (l *Logger) Debugf(f string, v ...interface{}) {
	if l.level <= LOG_DEBUG {
		l.LogD.Output(2, fmt.Sprintf(f, v...))
	}
	return
}

func (l *Logger) Info(v ...interface{}) {
	if l.level <= LOG_INFO {
		l.LogI.Output(2, fmt.Sprint(v...))
	}
	return
}

func (l *Logger) Infof(f string, v ...interface{}) {
	if l.level <= LOG_INFO {
		l.LogI.Output(2, fmt.Sprintf("\033[32m"+f+"\033[0m", v...))
	}
	return
}

func (l *Logger) Warn(v ...interface{}) {
	if l.level <= LOG_WARN {
		l.LogW.Output(2, fmt.Sprint(v...))
	}
	return
}

func (l *Logger) Warnf(f string, v ...interface{}) {
	if l.level <= LOG_WARN {
		l.LogW.Output(2, fmt.Sprintf("\033[33m"+f+"\033[0m", v...))
	}
	return
}

func (l *Logger) Error(v ...interface{}) {
	if l.level <= LOG_ERROR {
		l.LogE.Output(2, fmt.Sprint(v...))
	}
	return
}

func (l *Logger) Errorf(f string, v ...interface{}) {
	if l.level <= LOG_ERROR {
		l.LogE.Output(2, fmt.Sprintf("\033[31m"+f+"\033[0m", v...))
	}
	return
}

func (l *Logger) color(code int, level string) string {
	return fmt.Sprintf("\033[%dm%s %5s: \033[0m", code, l.prefix, level)
}
