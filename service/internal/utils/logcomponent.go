package utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
)

type PrefixFormatter struct {
	Prefix    string
	Formatter log.Formatter
}

func (f *PrefixFormatter) Format(entry *log.Entry) ([]byte, error) {
	entry.Message = fmt.Sprintf("%s %s", f.Prefix, entry.Message)

//	f.Formatter.(*log.TextFormatter).DisableTimestamp = true

	return f.Formatter.Format(entry)
}

type LogComponent struct {
	logger *log.Logger

	entry *log.Entry
}

func (c *LogComponent) SetPrefix(prefix string) {
	/*
	c.Logger().SetFormatter(&PrefixFormatter{
		Prefix:    "[" + prefix + "]",
//		Formatter: &log.TextFormatter{},
	})
	*/
	c.Logger();
	c.entry = c.logger.WithField("connector", prefix)
}

func (c *LogComponent) Logger() *log.Entry {
	if c.logger == nil {
		c.logger = log.New()
	}

	return c.entry
}
