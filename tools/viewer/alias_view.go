package dbviewer

import (
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type aliasView struct {
	GameID         string
	Source         schema.SourceID
	Method         string
	SourceLocation string
}

func (server *Server) aliasViews(item *schema.ItemDocument) []aliasView {
	aliases := make([]aliasView, 0, len(item.Aliases))
	for _, alias := range item.Aliases {
		gameID := "Unknown"
		if alias.GameID.Known {
			gameID = formatGameID(alias.GameID.Value)
		}
		source := server.sources[alias.GameID.Provenance.Source]
		aliases = append(aliases, aliasView{
			GameID:         gameID,
			Source:         alias.GameID.Provenance.Source,
			Method:         alias.GameID.Provenance.Method,
			SourceLocation: source.Location,
		})
	}
	return aliases
}
