package common

import (
	"context"
	"os"

	"github.com/amarin/logging"
)

// NewLoaderLogger returns the logger the download loaders (pkg/pymorphy,
// pkg/opencorpora, pkg/unimorph) embed. If the process-wide logger was
// configured with logging.Init before the call, it is a named logger at
// debug level; otherwise it is a logger that discards everything, so a
// library caller that never configures logging gets a working, silent
// loader instead of a panic. The choice is made once, at the call: a
// logging.Init made later does not affect an already created loader —
// assign its exported Logger field to change it.
func NewLoaderLogger(name string) (logger logging.Logger) {
	defer func() {
		// logging.NewNamedLogger panics when logging.Init was not called,
		// and the package offers no way to check that beforehand.
		if recover() != nil {
			logger = discardLogger{}
		}
	}()

	return logging.NewNamedLogger(name).WithLevel(logging.LevelDebug)
}

// discardLogger is a logging.Logger that drops every message.
type discardLogger struct{}

func (discardLogger) Level() logging.Level                 { return logging.LevelFatal }
func (discardLogger) IsEnabledForLevel(logging.Level) bool { return false }
func (discardLogger) Trace(...interface{})                 {}
func (discardLogger) Tracef(string, ...interface{})        {}
func (discardLogger) Debug(...interface{})                 {}
func (discardLogger) Debugf(string, ...interface{})        {}
func (discardLogger) Info(...interface{})                  {}
func (discardLogger) Infof(string, ...interface{})         {}
func (discardLogger) Warn(...interface{})                  {}
func (discardLogger) Warnf(string, ...interface{})         {}
func (discardLogger) Error(...interface{})                 {}
func (discardLogger) Errorf(string, ...interface{})        {}

// Fatal and Fatalf keep the logging.Logger contract: they exit with status 1.
func (discardLogger) Fatal(...interface{})                     { os.Exit(1) }
func (discardLogger) Fatalf(string, ...interface{})            { os.Exit(1) }
func (d discardLogger) WithKeys(logging.Keys) logging.Logger   { return d }
func (d discardLogger) WithKey(string, any) logging.Logger     { return d }
func (d discardLogger) WithError(error) logging.Logger         { return d }
func (d discardLogger) WithLevel(logging.Level) logging.Logger { return d }
func (d discardLogger) WithContext(context.Context) logging.Logger {
	return d
}
