package tft

import (
	"os"

	"github.com/sirupsen/logrus"
)

// NewLogger 初始化 logrus logger
//
// 环境变量：
//
//	LOG_LEVEL = debug | info | warn | error（默认 info）
//	LOG_ENV   = prod | dev（默认 dev，prod 输出 JSON，dev 输出带颜色的文本）
func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)

	// 日志级别
	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// 日志格式
	if os.Getenv("LOG_ENV") == "prod" {
		// 生产：JSON 格式，方便日志平台采集
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		// 开发：带颜色的文本，方便本地调试
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	return log
}
