package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var zapLogger *zap.Logger
var hostname string

func init() {
	var err error
	config := zap.NewProductionConfig()
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.StacktraceKey = "ERROR"
	config.EncoderConfig = encoderConfig
	zapLogger, err = config.Build(zap.AddCallerSkip(1))
	defer func(zapLogger *zap.Logger) {
		err := zapLogger.Sync()
		if err != nil {

		}
	}(zapLogger)
	hostname, _ = os.Hostname()
	if err != nil {
		panic(err)
	}
}

func Info(message string, fields ...zap.Field) {
	zapLogger.With(zap.String("hostname", hostname)).Info(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	zapLogger.With(zap.String("hostname", hostname)).Debug(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	zapLogger.With(zap.String("hostname", hostname)).Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	zapLogger.With(zap.String("hostname", hostname)).Fatal(message, fields...)
}
