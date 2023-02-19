package logger

import (
	"errors"

	"github.com/Shopify/sarama"
	logkafka "github.com/kenjones-cisco/logrus-kafka-hook"
	"github.com/sirupsen/logrus"
)

var LogKf *logrus.Logger

func DoInit(url string, topic string) (*logrus.Logger, error) {
	if url == "" || topic == "" {
		logrus.Errorf("url or topic 为空")
		return nil, errors.New("url or topic 为空")
	}
	producer, err := logkafka.SimpleProducer([]string{url}, sarama.CompressionSnappy, sarama.WaitForLocal, nil)
	if err != nil {
		logrus.Fatal(err)
	}
	hook := logkafka.New().WithFormatter(logkafka.DefaultFormatter(logrus.Fields{"appid": "rtc.server"})).WithProducer(producer).WithLevels([]logrus.Level{logrus.DebugLevel, logrus.ErrorLevel, logrus.InfoLevel, logrus.PanicLevel}).WithTopic(topic)
	LogKf = logrus.New()
	LogKf.SetLevel(logrus.ErrorLevel)
	LogKf.Hooks.Add(hook)
	return LogKf, nil
}

func SetLevel(l logrus.Level) {
	LogKf.SetLevel(l)
}

func Infof(format string, v ...interface{}) {
	if v == nil {
		LogKf.Infof(format, nil)
	} else {
		LogKf.Infof(format, v)
	}
}

func Debugf(format string, v ...interface{}) {
	if v == nil {
		LogKf.Debugf(format, nil)
	} else {
		LogKf.Debugf(format, v)
	}
}

func Warnf(format string, v ...interface{}) {
	if v == nil {
		LogKf.Warnf(format, nil)
	} else {
		LogKf.Warnf(format, v)
	}
}

func Errorf(format string, v ...interface{}) {
	if v == nil {
		LogKf.Errorf(format, nil)
	} else {
		LogKf.Errorf(format, v)
	}
}

func Panicf(format string, v ...interface{}) {
	if v == nil {
		LogKf.Panicf(format, nil)
	} else {
		LogKf.Panicf(format, v)
	}
}
