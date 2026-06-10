package config

import (
	"github.com/sirupsen/logrus"
	librespot "github.com/devgianlu/go-librespot"
)

// LogrusAdapter adapts logrus.Logger to librespot.Logger interface
type LogrusAdapter struct {
	logger *logrus.Logger
	fields logrus.Fields
}

// NewLogrusAdapter creates a new adapter
func NewLogrusAdapter(logger *logrus.Logger) *LogrusAdapter {
	return &LogrusAdapter{
		logger: logger,
		fields: make(logrus.Fields),
	}
}

func (a *LogrusAdapter) Tracef(format string, args ...interface{}) {
	a.logger.WithFields(a.fields).Tracef(format, args...)
}

func (a *LogrusAdapter) Debugf(format string, args ...interface{}) {
	a.logger.WithFields(a.fields).Debugf(format, args...)
}

func (a *LogrusAdapter) Infof(format string, args ...interface{}) {
	a.logger.WithFields(a.fields).Infof(format, args...)
}

func (a *LogrusAdapter) Warnf(format string, args ...interface{}) {
	a.logger.WithFields(a.fields).Warnf(format, args...)
}

func (a *LogrusAdapter) Errorf(format string, args ...interface{}) {
	a.logger.WithFields(a.fields).Errorf(format, args...)
}

func (a *LogrusAdapter) Trace(args ...interface{}) {
	a.logger.WithFields(a.fields).Trace(args...)
}

func (a *LogrusAdapter) Debug(args ...interface{}) {
	a.logger.WithFields(a.fields).Debug(args...)
}

func (a *LogrusAdapter) Info(args ...interface{}) {
	a.logger.WithFields(a.fields).Info(args...)
}

func (a *LogrusAdapter) Warn(args ...interface{}) {
	a.logger.WithFields(a.fields).Warn(args...)
}

func (a *LogrusAdapter) Error(args ...interface{}) {
	a.logger.WithFields(a.fields).Error(args...)
}

func (a *LogrusAdapter) WithField(key string, value interface{}) librespot.Logger {
	newFields := make(logrus.Fields)
	for k, v := range a.fields {
		newFields[k] = v
	}
	newFields[key] = value
	return &LogrusAdapter{
		logger: a.logger,
		fields: newFields,
	}
}

func (a *LogrusAdapter) WithError(err error) librespot.Logger {
	return a.WithField("error", err)
}
