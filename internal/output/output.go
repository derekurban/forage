package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/derekurban/forage/internal/apperr"
)

type Options struct {
	JSON    bool
	JSONL   bool
	NoColor bool
	Verbose bool
}

type Envelope struct {
	OK          bool          `json:"ok"`
	Command     string        `json:"command"`
	GeneratedAt string        `json:"generated_at"`
	Data        any           `json:"data,omitempty"`
	Diagnostics any           `json:"diagnostics,omitempty"`
	Error       *apperr.Error `json:"error,omitempty"`
}

func Write(w io.Writer, opts Options, command string, data any, diagnostics any) error {
	if opts.JSON || opts.JSONL {
		env := Envelope{OK: true, Command: command, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Data: data, Diagnostics: diagnostics}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if opts.JSONL {
			enc.SetIndent("", "")
		}
		return enc.Encode(env)
	}
	return nil
}

func WriteError(w io.Writer, opts Options, command string, err *apperr.Error) error {
	if opts.JSON || opts.JSONL {
		env := Envelope{OK: false, Command: command, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Error: err}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if opts.JSONL {
			enc.SetIndent("", "")
		}
		return enc.Encode(env)
	}
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	if opts.NoColor {
		style = lipgloss.NewStyle().Bold(true)
	}
	_, werr := fmt.Fprintf(w, "%s %s\n", style.Render("Error:"), err.Message)
	return werr
}

func Heading(opts Options, s string) string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	if opts.NoColor {
		style = lipgloss.NewStyle().Bold(true)
	}
	return style.Render(s)
}

func Status(opts Options, s string) string {
	color := "8"
	switch strings.ToLower(s) {
	case "healthy", "configured", "ok", "implemented":
		color = "2"
	case "missing", "disabled", "exhausted", "error":
		color = "9"
	case "metadata_only", "cooldown", "degraded", "not_configured":
		color = "11"
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	if opts.NoColor {
		style = lipgloss.NewStyle()
	}
	return style.Render(s)
}
