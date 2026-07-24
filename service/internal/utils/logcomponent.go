package utils

import (
	log "github.com/sirupsen/logrus"
)

type LogComponent struct {
	entry *log.Entry
}

func (c *LogComponent) SetPrefix(prefix string) {
	c.entry = log.WithField("connector", prefix)
}

func (c *LogComponent) Logger() *log.Entry {
	if c.entry == nil {
		c.entry = log.WithFields(log.Fields{})
	}
	return c.entry
}
