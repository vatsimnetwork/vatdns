package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zapLogger *zap.Logger

func init() {
	var err error
	config := zap.NewProductionConfig()
	encoderConfig := zap.NewProductionEncoderConfig()
	zapcore.TimeEncoderOfLayout("Jan _2 15:04:05.000000000")
	encoderConfig.StacktraceKey = "" // to hide stacktrace info
	config.EncoderConfig = encoderConfig

	zapLogger, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
}

func Info(message string, fields ...zap.Field) {
	zapLogger.Info(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	zapLogger.Debug(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	zapLogger.Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	zapLogger.Fatal(message, fields...)
}
