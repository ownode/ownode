package config

import (
    "github.com/Sirupsen/logrus"
    "os"
    "fmt"
)

var log *CustomLog

// custom log supports prepending logs with a name or identifier
type CustomLog struct {
	BaseLogger *logrus.Logger
	Name string
	Id string
	EnableNaming bool
}

func (cl *CustomLog) makeName() string {
	return fmt.Sprintf("%s-%s: ", cl.Name, cl.Id)
}

func (cl *CustomLog) Info(args... interface{}) {
	if cl.EnableNaming {
		args = append([]interface{}{ cl.makeName() }, args...)
	}
	cl.BaseLogger.Info(args...)
}

func (cl *CustomLog) Debug(args... interface{}) {
	if cl.EnableNaming {
		args = append([]interface{}{ cl.makeName() }, args...)
	}
	cl.BaseLogger.Debug(args...)
}

func (cl *CustomLog) Error(args... interface{}) {
	if cl.EnableNaming {
		args = append([]interface{}{ cl.makeName() }, args...)
	}
	cl.BaseLogger.Error(args...)
}

func (cl *CustomLog) Warn(args... interface{}) {
	if cl.EnableNaming {
		args = append([]interface{}{ cl.makeName() }, args...)
	}
	cl.BaseLogger.Warn(args...)
}

// Set log name
func (cl *CustomLog) SetName(name, id string) {
	cl.Name = name
	cl.Id = id
}

func init(){
	log = &CustomLog{ BaseLogger: logrus.New() }
}

// return a nameless log
func Log() *CustomLog {
    log.EnableNaming = false
    // logrus.SetFormatter(&logrus.JSONFormatter{})
	log.BaseLogger.Out = os.Stderr
  	log.BaseLogger.Level = logrus.DebugLevel
	return log
}

func LogWithName(name string, id string) *CustomLog {
	log.EnableNaming = true
	log.SetName(name, id)
	// logrus.SetFormatter(&logrus.JSONFormatter{})
	log.BaseLogger.Out = os.Stderr
  	log.BaseLogger.Level = logrus.DebugLevel
	return log
}


