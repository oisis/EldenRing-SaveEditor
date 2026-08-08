package dbviewer

import (
	"fmt"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// sourceOriginClass maps a provenance source to the CSS class that colours the
// rendered source text. It describes origin only, never data quality. Other
// sources keep the surrounding text colour, so they get no class. Templates
// reach it through the "sourceOrigin" function, so every view type shares this
// one classification.
func sourceOriginClass(source schema.SourceID) string {
	switch {
	case source == "legacy_db_data":
		return "legacy"
	case strings.HasPrefix(string(source), "regulation_"):
		return "regulation"
	default:
		return ""
	}
}

type factView struct {
	Label          string
	Value          string
	Known          bool
	NotApplicable  bool
	Source         schema.SourceID
	Method         string
	SourceLocation string
}

func (server *Server) fact(label string, known bool, value any, provenance schema.Provenance) factView {
	displayValue := "Unknown"
	if known {
		displayValue = fmt.Sprint(value)
	}
	source := server.sources[provenance.Source]
	return factView{
		Label:          label,
		Value:          displayValue,
		Known:          known,
		Source:         provenance.Source,
		Method:         provenance.Method,
		SourceLocation: source.Location,
	}
}
