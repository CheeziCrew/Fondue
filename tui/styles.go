package tui

import "github.com/charmbracelet/lipgloss"

// ── Theme ───────────────────────────────────────────────────────────
// All colors flow through Theme so swapping palettes is trivial.
// Uses ANSI 0-15 so colors follow the terminal theme (base16 via Tinty etc).

type Theme struct {
	Bg      lipgloss.Color
	Fg      lipgloss.Color
	Dim     lipgloss.Color
	Red     lipgloss.Color
	Green   lipgloss.Color
	Yellow  lipgloss.Color
	Blue    lipgloss.Color
	Magenta lipgloss.Color
	Cyan    lipgloss.Color
	Accent  lipgloss.Color
	BrCyan  lipgloss.Color
	BrWhite lipgloss.Color
}

// DefaultTheme uses ANSI 0-15, adapts to terminal color scheme.
var DefaultTheme = Theme{
	Bg:      lipgloss.Color("0"),
	Fg:      lipgloss.Color("7"),
	Dim:     lipgloss.Color("8"),
	Red:     lipgloss.Color("1"),
	Green:   lipgloss.Color("2"),
	Yellow:  lipgloss.Color("3"),
	Blue:    lipgloss.Color("4"),
	Magenta: lipgloss.Color("5"),
	Cyan:    lipgloss.Color("6"),
	Accent:  lipgloss.Color("13"),
	BrCyan:  lipgloss.Color("14"),
	BrWhite: lipgloss.Color("15"),
}

// Active theme — change this to swap palettes.
var t = DefaultTheme

// ── Derived color vars (for backward compat in rendering code) ─────
var (
	colorBg      = t.Bg
	colorRed     = t.Red
	colorGreen   = t.Green
	colorYellow  = t.Yellow
	colorBlue    = t.Blue
	colorMagenta = t.Magenta
	colorCyan    = t.Cyan
	colorFg      = t.Fg
	colorDim     = t.Dim
	colorAccent  = t.Accent
	colorBrCyan  = t.BrCyan
	colorBrWhite = t.BrWhite
)

// ── List view styles ────────────────────────────────────────────────
var (
	listTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Accent).
			Padding(0, 2).
			MarginBottom(1)

	dimStyle = lipgloss.NewStyle().
			Foreground(t.Dim)
)

// ── Detail view styles ──────────────────────────────────────────────
var (
	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Magenta).
			Padding(1, 2).
			MarginTop(1).
			MarginLeft(2).
			MarginRight(2)

	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(t.Accent).
				Padding(0, 1).
				MarginBottom(1)

	detailPathStyle = lipgloss.NewStyle().
			Foreground(t.Dim).
			Italic(true).
			PaddingLeft(1)

	badgeOutStyle = lipgloss.NewStyle().
			Foreground(t.Bg).
			Background(t.Magenta).
			Bold(true).
			Padding(0, 1)

	badgeInStyle = lipgloss.NewStyle().
			Foreground(t.Bg).
			Background(t.Green).
			Bold(true).
			Padding(0, 1)

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(t.Cyan).
				PaddingTop(1).
				PaddingBottom(0).
				PaddingLeft(1)

	sectionCountStyle = lipgloss.NewStyle().
				Foreground(t.Dim)

	internalStyle = lipgloss.NewStyle().
			Foreground(t.Green)

	externalStyle = lipgloss.NewStyle().
			Foreground(t.Yellow)

	externalTagStyle = lipgloss.NewStyle().
				Foreground(t.Red).
				Bold(true)

	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Accent)

	noneStyle = lipgloss.NewStyle().
			Foreground(t.Dim).
			Italic(true).
			PaddingLeft(3)

	hintBarStyle = lipgloss.NewStyle().
			Foreground(t.Dim).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(t.Dim).
			PaddingTop(0).
			MarginTop(1).
			PaddingLeft(1)

	hintKeyStyle = lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true)

	hintSepStyle = lipgloss.NewStyle().
			Foreground(t.Dim)

	hintDescStyle = lipgloss.NewStyle().
			Foreground(t.Fg)

	arrowOutStyle = lipgloss.NewStyle().
			Foreground(t.Magenta).
			Bold(true)

	arrowInStyle = lipgloss.NewStyle().
			Foreground(t.Green).
			Bold(true)

	staleStyle = lipgloss.NewStyle().
			Foreground(t.Yellow).
			Bold(true)

	versionMatchStyle = lipgloss.NewStyle().
				Foreground(t.Green)
)
