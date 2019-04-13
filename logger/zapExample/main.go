package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func syslogTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(fmt.Sprintf("| %-8d|:", t.Unix()))
}

func customLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func customEncodeCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	// enc.AppendString(fmt.Sprintf("%d", caller.Line))
	enc.AppendString("TTTest")
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()
	logger.Info("failed to fetch URL",
		// Structured context as strongly typed Field values.
		zap.String("url", "www.google.com"),
		zap.Int("attempt", 3),
		zap.Duration("backoff", time.Second),
	)

	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	topicDebugging := zapcore.AddSync(ioutil.Discard)
	topicErrors := zapcore.AddSync(ioutil.Discard)

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	kafkaEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	core := zapcore.NewTee(
		zapcore.NewCore(kafkaEncoder, topicErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(kafkaEncoder, topicDebugging, lowPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
	)

	logger = zap.New(core)
	defer logger.Sync()
	logger.Error("found error ")
	logger.Info("constructed a logger")

	rawJSON := []byte(`{
		"level": "error",
		"encoding": "console",
		"outputPaths": ["stdout", "/tmp/logs"],
		"errorOutputPaths": ["stderr"],
		"encoderConfig": {
			"timeKey": "time",
			"messageKey": "message",
		  	"levelKey": "level",
			"callerEncoder":"callerEncoder",
			"levelEncoder": "lowercase"
		}
	}`)

	var cfg zap.Config

	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	cfg.DisableCaller = false
	cfg.EncoderConfig.EncodeTime = syslogTimeEncoder
	cfg.EncoderConfig.EncodeLevel = customLevelEncoder
	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	logger2, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger2.Sync()

	logger2.Info("logger2 construction succeeded")
	logger2.Error("second error ...")
}
