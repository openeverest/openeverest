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

// Package output provides utilities to print output in commands.
package output

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/commands/common"
)

// PrintError formats and prints an error to logger.
func PrintError(err error, l *zap.SugaredLogger, prettyPrint bool) {
	if errors.Is(err, common.ErrExitWithError) {
		return
	}

	if prettyPrint {
		_, _ = fmt.Fprintln(os.Stderr, Failure("%s", err))
	} else {
		l.Error(err)
	}
}

//nolint:gochecknoglobals
var (
	// Style is applied to the successful result.
	okStyle = lipgloss.NewStyle().
		Foreground(
			lipgloss.AdaptiveColor{Light: "#000000", Dark: "#5fd700"},
		)
	okStatus = okStyle.Render("✅")

	// Style is applied to the failure result.
	failureStyle = lipgloss.NewStyle().
			Foreground(
			lipgloss.AdaptiveColor{Light: "#B10810", Dark: "#F37C6F"},
		)
	failStatus = failureStyle.Render("❌")
)

// Success prints a message with a success emoji.
func Success(msg string, args ...any) string {
	return fmt.Sprintf("%s %s\n", okStatus, fmt.Sprintf(msg, args...))
}

// Failure prints a message with a fail emoji.
func Failure(msg string, args ...any) string {
	// return fmt.Sprintf("%s %s\n", failStatus, fmt.Sprintf(msg, args...))
	return fmt.Sprintf("%s %s\n",
		failStatus,
		failureStyle.Render(fmt.Sprintf(msg, args...)),
	)
}

// Info prints a message with an info emoji.
func Info(msg string, args ...any) string {
	return fmt.Sprintf("ℹ️  %s\n", fmt.Sprintf(msg, args...))
}

// Rocket prints a message with a rocket emoji.
func Rocket(msg string, args ...any) string {
	return fmt.Sprintf("🚀 %s\n", fmt.Sprintf(msg, args...))
}

// Warn prints a message with a warning emoji.
func Warn(msg string, args ...any) string {
	return fmt.Sprintf("⚠️  %s\n", fmt.Sprintf(msg, args...))
}

// Numeric prints a message with a numeric emoji.
func Numeric(num int, msg string, args ...any) string {
	return fmt.Sprintf("%d %s  %s\n", num, "\u2B95", fmt.Sprintf(msg, args...))
}
