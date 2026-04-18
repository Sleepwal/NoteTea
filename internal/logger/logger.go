package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	infoLog  *log.Logger
	errorLog *log.Logger
	debugLog *log.Logger
	enabled  bool
)

func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	logDir := filepath.Join(home, ".agenttea", "logs")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("agenttea_%s.log", time.Now().Format("2006-01-02")))

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	infoLog = log.New(f, "[INFO]  ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLog = log.New(f, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
	debugLog = log.New(f, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	enabled = true

	return nil
}

func InitWithWriter(w io.Writer) {
	infoLog = log.New(w, "[INFO]  ", log.Ldate|log.Ltime)
	errorLog = log.New(w, "[ERROR] ", log.Ldate|log.Ltime)
	debugLog = log.New(w, "[DEBUG] ", log.Ldate|log.Ltime)
	enabled = true
}

func Info(format string, v ...interface{}) {
	if enabled && infoLog != nil {
		infoLog.Output(2, fmt.Sprintf(format, v...))
	}
}

func Error(format string, v ...interface{}) {
	if enabled && errorLog != nil {
		errorLog.Output(2, fmt.Sprintf(format, v...))
	}
}

func Debug(format string, v ...interface{}) {
	if enabled && debugLog != nil {
		debugLog.Output(2, fmt.Sprintf(format, v...))
	}
}
