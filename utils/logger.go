package utils

import (
    "log"
    "os"
    "sync"
)

type LogLevelType int

const (
    ERROR LogLevelType = iota
    WARNING
    INFO
    DEBUG
)

const (
    logName string = "tagger.log"
)

var (
    mutex sync.Mutex
    logLevel LogLevelType = INFO
    logger *log.Logger = nil
)

func LogLevel() LogLevelType {
    mutex.Lock()
    defer mutex.Unlock()

    return logLevel
}

func SetLogLevel(level LogLevelType) {
    mutex.Lock()
    defer mutex.Unlock()

    logLevel = level
}

func Log(level LogLevelType, msg string, args ... interface{}) {
    mutex.Lock()
    defer mutex.Unlock()

    if level > logLevel {
        return
    }

    if logger == nil {
        createLogger()
        if logger == nil {
            return
        }
    }

    logger.Printf(prefix[level] + msg, args ...)
}

func createLogger() {
    file, err := os.OpenFile(logName, os.O_RDWR | os.O_CREATE | os.O_TRUNC, 0666)
    if err != nil {
        log.Fatal("error opening file '%v': %v", logName, err)
    }
    
    logger = log.New(file, "", log.LstdFlags)
}

var prefix = map[LogLevelType]string {
    ERROR: "ERROR ", WARNING: "WARNING ", INFO: "INFO ", DEBUG: "DEBUG ",
}
