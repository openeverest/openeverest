// everest
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package logger provides functionality related to logging.
package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/openeverest/openeverest/v2/cmd/config"
)

// MustInitLogger initializes logger and panics in case of an error. // FIXME Test this.
func MustInitLogger(json bool, logName string) *zap.Logger {
	var loggerCfg zap.Config
	if config.Debug {
		loggerCfg = zap.NewDevelopmentConfig()
		loggerCfg.DisableCaller = true
	} else {
		loggerCfg = zap.NewProductionConfig()
		loggerCfg.EncoderConfig = zap.NewProductionEncoderConfig()
		loggerCfg.EncoderConfig.TimeKey = "T"
		loggerCfg.EncoderConfig.EncodeTime = zapcore.TimeEncoder(func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.UTC().Format("2006-01-02T15:04:05Z0700"))
		})
	}
	loggerCfg.Encoding = "console"
	if json {
		loggerCfg.Encoding = "json"
	}
	loggerCfg.DisableStacktrace = true

	return initGlobalLogger(loggerCfg, logName)
}

// MustInitVerboseLogger initializes a verbose logger and panics in case of an error.
func MustInitVerboseLogger(json bool, logName string) *zap.Logger {
	lCfg := zap.NewDevelopmentConfig()
	if json {
		lCfg.Encoding = "json"
	}
	return initGlobalLogger(lCfg, logName)
}

// InitLoggerInRootCmd inits the provided logger instance based on command's flags.
// This is meant to be run by the root command in PersistentPreRun step.
func InitLoggerInRootCmd(verbose, json bool, logName string) {
	if verbose {
		MustInitVerboseLogger(json, logName)
	} else {
		MustInitLogger(json, logName)
	}
}

// GetLogger returns the global logger instance.
func GetLogger() *zap.SugaredLogger {
	return zap.L().Sugar()
}

func initGlobalLogger(loggerCfg zap.Config, logName string) *zap.Logger {
	logger, err := loggerCfg.Build()
	if err != nil {
		panic(fmt.Sprintf("Cannot initialize logger: %s", err))
	}

	zap.ReplaceGlobals(logger.Named(logName))

	return zap.L()
}
