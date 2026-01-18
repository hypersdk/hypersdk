// SPDX-License-Identifier: LGPL-3.0-or-later

package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

type StandardLogger struct {
	level  Level
	logger *log.Logger
}

func New(levelStr string) Logger {
	level := INFO
	switch strings.ToLower(levelStr) {
	case "debug":
		level = DEBUG
	case "info":
		level = INFO
	case "warn", "warning":
		level = WARN
	case "error":
		level = ERROR
	}

	return &StandardLogger{
		level:  level,
		logger: log.New(os.Stderr, "", 0),
	}
}

func (l *StandardLogger) log(level Level, levelStr, msg string, keysAndValues ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("[%s] %s: %s", timestamp, levelStr, msg)

	if len(keysAndValues) > 0 {
		var pairs []string
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				pairs = append(pairs, fmt.Sprintf("%v=%v", keysAndValues[i], keysAndValues[i+1]))
			}
		}
		if len(pairs) > 0 {
			prefix = fmt.Sprintf("%s | %s", prefix, strings.Join(pairs, ", "))
		}
	}

	l.logger.Println(prefix)
}

func (l *StandardLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.log(DEBUG, "DEBUG", msg, keysAndValues...)
}

func (l *StandardLogger) Info(msg string, keysAndValues ...interface{}) {
	l.log(INFO, "INFO", msg, keysAndValues...)
}

func (l *StandardLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.log(WARN, "WARN", msg, keysAndValues...)
}

func (l *StandardLogger) Error(msg string, keysAndValues ...interface{}) {
	l.log(ERROR, "ERROR", msg, keysAndValues...)
}
