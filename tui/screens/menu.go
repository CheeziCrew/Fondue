package screens

import (
	"github.com/CheeziCrew/curd"
)

// MenuModel wraps curd.MenuModel for fondue.
type MenuModel = curd.MenuModel

// NewMenu creates a fresh menu model.
func NewMenu() curd.MenuModel {
	return curd.NewMenuModel(curd.MenuConfig{
		Banner: []string{
			"   ___              __        ",
			"  / _/__  ___  ____/ /_ _____ ",
			" / _/ _ \\/ _ \\/ _  / // / -_)",
			"/_/ \\___/_//_/\\_,_/\\_,_/\\__/ ",
		},
		Tagline: "melt through your microservice dependencies",
		Palette: curd.FonduePalette,
		Items: []curd.MenuItem{
			{Icon: "🔍", Name: "Explore", Command: "explore", Desc: "interactive dependency graph"},
			{Icon: "📊", Name: "Stats", Command: "stats", Desc: "service counts & connectivity"},
			{Icon: "⚠️", Name: "Stale Specs", Command: "stale", Desc: "outdated integration specs"},
		},
	})
}
