package logging

import (
	"campron_enterprise/backend/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger = zap.Logger

func New(cfg *config.Config) *zap.Logger {
	// Production-friendly JSON logs
	c := zap.NewProductionConfig()
	c.EncoderConfig.TimeKey = "ts"
	c.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := c.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

func Str(k, v string) zap.Field { return zap.String(k, v) }
func Int(k string, v int) zap.Field { return zap.Int(k, v) }
func Err(err error) zap.Field { return zap.Error(err) }
