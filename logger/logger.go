package logger

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var _logger *zap.SugaredLogger

func init() {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}
	// 同时输出到文件和标准输出中
	w := io.MultiWriter(lumberJackLogger, os.Stdout)
	writeSyncer := zapcore.AddSync(w)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	_logger = logger.Sugar()
}

func Debug(args ...interface{}) {
	_logger.Debug(args...)
}

func Info(args ...interface{}) {
	_logger.Info(args...)
}

func Warn(args ...interface{}) {
	_logger.Warn(args...)
}

func Error(args ...interface{}) {
	_logger.Error(args...)
}

func Panic(args ...interface{}) {
	_logger.Panic(args...)
}

func Fatal(args ...interface{}) {
	_logger.Fatal(args...)
}

func Debugf(template string, args ...interface{}) {
	_logger.Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	_logger.Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	_logger.Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	_logger.Errorf(template, args...)
}

func Panicf(template string, args ...interface{}) {
	_logger.Panicf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	_logger.Fatalf(template, args...)
}
