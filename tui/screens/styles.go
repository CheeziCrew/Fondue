package screens

import (
	"charm.land/lipgloss/v2"
	"github.com/CheeziCrew/curd"
)

// Fondue uses the green/orange palette.
var (
	palette = curd.FonduePalette
	st      = palette.Styles()
)

// Base16 colors from curd.
var (
	colorBg      = curd.ColorBg
	colorRed     = curd.ColorRed
	colorGreen   = curd.ColorGreen
	colorYellow  = curd.ColorYellow
	colorCyan    = curd.ColorCyan
	colorFg      = curd.ColorFg
	colorDim     = curd.ColorGray
	colorAccent  = curd.ColorBrGreen
)

// ── Shared styles ───────────────────────────────────────────────────

var dimStyle = lipgloss.NewStyle().Foreground(colorDim)

// ── Detail view styles ──────────────────────────────────────────────

var (
	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Padding(1, 2).
			MarginTop(1).
			MarginLeft(2).
			MarginRight(2)

	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent).
				Padding(0, 1).
				MarginBottom(1)

	detailPathStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true).
			PaddingLeft(1)

	badgeOutStyle = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorYellow).
			Bold(true).
			Padding(0, 1)

	badgeInStyle = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorGreen).
			Bold(true).
			Padding(0, 1)

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorCyan).
				PaddingTop(1).
				PaddingBottom(0).
				PaddingLeft(1)

	sectionCountStyle = lipgloss.NewStyle().
				Foreground(colorDim)

	internalStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	externalStyle = lipgloss.NewStyle().
			Foreground(colorYellow)

	externalTagStyle = lipgloss.NewStyle().
				Foreground(colorRed).
				Bold(true)

	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	noneStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true).
			PaddingLeft(3)

	arrowOutStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	arrowInStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	staleStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	versionMatchStyle = lipgloss.NewStyle().
				Foreground(colorGreen)
)
