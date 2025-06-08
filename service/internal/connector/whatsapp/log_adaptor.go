package whatsapp

import (
	"github.com/sirupsen/logrus"
	walog "go.mau.fi/whatsmeow/util/log"
)

type WaLogAdaptor struct {
	Logger *logrus.Logger
}

func (a WaLogAdaptor) Warnf(msg string, args ...any) {
	a.Logger.Warnf(msg, args...)
}

func (a WaLogAdaptor) Debugf(msg string, args ...any) {
	a.Logger.Debugf(msg, args...)
}

func (a WaLogAdaptor) Errorf(msg string, args ...any) {
	a.Logger.Errorf(msg, args...)
}

func (a WaLogAdaptor) Infof(msg string, args ...any) {
	a.Logger.Infof(msg, args...)
}

func (a WaLogAdaptor) Sub(module string) walog.Logger {
	a.Logger.Infof("sub: " + module)

	return a
}
