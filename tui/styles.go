package tui

import "github.com/charmbracelet/lipgloss"

// ── Color palette ───────────────────────────────────────────────────
// Uses ANSI 0-15 so colors follow the terminal theme (base16 via Tinty etc).
//
//	0  background     8  bright black (comments/dim)
//	1  red            9  bright red
//	2  green         10  (base01)
//	3  yellow        11  (base02)
//	4  blue          12  (base04)
//	5  magenta       13  bright magenta
//	6  cyan          14  bright cyan
//	7  foreground    15  bright white
var (
	colorBg      = lipgloss.Color("0")
	colorRed     = lipgloss.Color("1")
	colorGreen   = lipgloss.Color("2")
	colorYellow  = lipgloss.Color("3")
	colorBlue    = lipgloss.Color("4")
	colorMagenta = lipgloss.Color("5")
	colorCyan    = lipgloss.Color("6")
	colorFg      = lipgloss.Color("7")
	colorDim     = lipgloss.Color("8")
	colorAccent  = lipgloss.Color("13") // bright magenta — primary accent
	colorBrCyan  = lipgloss.Color("14")
	colorBrWhite = lipgloss.Color("15")
)

// ── List view styles ────────────────────────────────────────────────
var (
	listTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Padding(0, 2).
			MarginBottom(1)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorDim)
)

// ── Detail view styles ──────────────────────────────────────────────
var (
	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMagenta).
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

	// Detail view badges — we own the full render so background colors are safe.
	badgeOutStyle = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorMagenta).
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

	hintBarStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(colorDim).
			PaddingTop(0).
			MarginTop(1).
			PaddingLeft(1)

	hintKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	hintSepStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	hintDescStyle = lipgloss.NewStyle().
			Foreground(colorFg)

	arrowOutStyle = lipgloss.NewStyle().
			Foreground(colorMagenta).
			Bold(true)

	arrowInStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	staleStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	versionMatchStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	versionDimStyle = lipgloss.NewStyle().
			Foreground(colorDim)
)
