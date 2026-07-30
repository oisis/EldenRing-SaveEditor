package dbviewer

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type factView struct {
	Label          string
	Value          string
	Known          bool
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
