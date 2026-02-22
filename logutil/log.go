package logutil

import "github.com/fatih/color"

// Color functions for logging
var (
	Success = color.New(color.FgGreen).SprintfFunc()
	Info    = color.New(color.FgCyan).SprintfFunc()
	Warn    = color.New(color.FgYellow).SprintfFunc()
	Error   = color.New(color.FgRed).SprintfFunc()
	Debug   = color.New(color.FgBlue).SprintfFunc()
	Remote  = color.New(color.FgMagenta).SprintfFunc()
	Muted   = color.New(color.Faint).SprintfFunc()
)
