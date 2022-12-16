package mocks

import (
	"errors"
	"fmt"
	"strings"
)

// Logger is used to implement logging functions for testing the
// Kubernetes Secrets storage provider.
type Logger struct {
	errors   []string
	warnings []string
	infos    []string
	debugs   []string
}

// NewLogger returns a shiny, new Logger
func NewLogger() *Logger {
	return &Logger{}
}

// RecordedError logs that an error has occurred and returns a new error
// with the given error message.
func (l *Logger) RecordedError(msg string, args ...interface{}) error {
	errStr := fmt.Sprintf(msg, args...)
	l.errors = append(l.errors, errStr)
	return errors.New(errStr)
}

// Error logs an error.
func (l *Logger) Error(msg string, args ...interface{}) {
	l.errors = append(l.errors, fmt.Sprintf(msg, args...))
}

// Warn logs a warning.
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.warnings = append(l.warnings, fmt.Sprintf(msg, args...))
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...interface{}) {
	l.infos = append(l.infos, fmt.Sprintf(msg, args...))
}

// ClearInfo Clears the info messages
func (l *Logger) ClearInfo() {
	l.infos = nil
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.debugs = append(l.debugs, fmt.Sprintf(msg, args...))
}

func (l *Logger) messageWasLogged(msg string, loggedMsgs []string) bool {
	for _, loggedMsg := range loggedMsgs {
		if strings.Contains(loggedMsg, msg) {
			return true
		}
	}
	return false
}

// ErrorWasLogged determines if an error string appears in any
// errors that have been logged.
func (l *Logger) ErrorWasLogged(errStr string) bool {
	return l.messageWasLogged(errStr, l.errors)
}

// WarningWasLogged determines if a warning string appears in any
// warning messages that have been logged.
func (l *Logger) WarningWasLogged(warning string) bool {
	return l.messageWasLogged(warning, l.warnings)
}

// InfoWasLogged determines if a warning string appears in any
// info messages that have been logged.
func (l *Logger) InfoWasLogged(info string) bool {
	return l.messageWasLogged(info, l.infos)
}

// DebugWasLogged determines if a debug string appears in any
// debug messages that have been logged.
func (l *Logger) DebugWasLogged(debug string) bool {
	return l.messageWasLogged(debug, l.debugs)
}
