package dbviewer

import (
	"net/http"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type itemPage struct {
	Meta           pageMeta
	Name           string
	IconURL        string
	GameID         string
	GameIDPath     string
	Key            string
	EntryType      string
	DocumentPath   string
	Identity       []factView
	Presentation   []factView
	TextMetadata   []factView
	Storage        []factView
	Safety         []factView
	Acquisition    []factView
	Modifiers      []factView
	Links          []factView
	Unlocks        []factView
	TechnicalData  []factView
	Capabilities   []capabilityView
	FamilyData     []factView
	Variants       []variantView
	Aliases        []aliasView
	SourceRecords  []parameterRecordView
	Relations      []relationView
	UnknownCount   int
	HasDescription bool
}

func (server *Server) itemHandler(response http.ResponseWriter, request *http.Request) {
	gameID, err := parseGameID(request.PathValue("gameID"))
	if err != nil {
		http.NotFound(response, request)
		return
	}
	view, exists := server.catalog.ItemViewByGameID(gameID)
	if !exists {
		http.NotFound(response, request)
		return
	}
	document, exists := server.documentsByID[view.Resource.ID]
	if !exists {
		http.Error(response, "Catalog document is unavailable.", http.StatusInternalServerError)
		return
	}
	server.render(response, "item", server.buildItemPage(view, document, gameID))
}

func (server *Server) buildItemPage(
	view gamecatalog.ItemView,
	document loader.Document,
	requestedGameID uint32,
) itemPage {
	resource := view.Resource
	item := resource.Item
	name := resource.Item.Presentation.DisplayName.Value
	iconURL := itemIconURL(item)
	entryType := "Canonical"
	unknownCount := countUnknownFacts(resource)
	gameIDKnown := item.GameID.Known
	gameIDProvenance := item.GameID.Provenance
	categoryKnown, categoryValue, categoryProvenance := item.Category.Known, item.Category.Value, item.Category.Provenance
	subcategoryKnown, subcategoryValue, subcategoryProvenance := item.Subcategory.Known, item.Subcategory.Value, item.Subcategory.Provenance
	presentation := item.Presentation
	storage := item.Storage
	safety := item.Safety
	acquisition := item.Acquisition
	modifiers := item.Modifiers
	links := item.Links
	unlocks := item.Unlocks
	technicalData := item.RelatedTechnicalRecords
	capabilities := item.Capabilities
	familyData := server.familyFacts(item)
	sourceRecords := server.parameterRecordViews(item)
	if variant, exists := findVariant(item, requestedGameID); exists {
		name = variantDisplayName(resource.Item.Presentation.DisplayName.Value, variant)
		iconURL = variantIconURL(item, variant)
		entryType = "Variant"
		unknownCount = countUnknownFacts(variant)
		gameIDKnown = variant.GameID.Known
		gameIDProvenance = variant.GameID.Provenance
		categoryKnown, categoryValue, categoryProvenance = variant.Data.Category.Known, variant.Data.Category.Value, variant.Data.Category.Provenance
		subcategoryKnown, subcategoryValue, subcategoryProvenance = variant.Data.Subcategory.Known, variant.Data.Subcategory.Value, variant.Data.Subcategory.Provenance
		presentation = variant.Data.Presentation
		storage = variant.Data.Storage
		safety = variant.Data.Safety
		acquisition = variant.Data.Acquisition
		modifiers = variant.Data.Modifiers
		links = variant.Data.Links
		unlocks = variant.Data.Unlocks
		technicalData = variant.Data.RelatedTechnicalRecords
		capabilities = variant.Data.Capabilities
		familyData = server.variantFamilyFacts(item.Family.Value, variant.Data)
		sourceRecords = server.parameterRecords(
			"Variant "+formatGameID(variant.GameID.Value),
			variant.SourceRecords,
		)
	}
	gameID := formatGameID(requestedGameID)
	return itemPage{
		Meta:         server.pageMeta(name),
		Name:         name,
		IconURL:      iconURL,
		GameID:       gameID,
		GameIDPath:   strings.TrimPrefix(gameID, "0x"),
		Key:          resource.Key,
		EntryType:    entryType,
		DocumentPath: document.Path,
		Identity: []factView{
			server.fact("Game ID", gameIDKnown, gameID, gameIDProvenance),
			server.fact("Family", item.Family.Known, item.Family.Value, item.Family.Provenance),
			server.fact("Category", categoryKnown, categoryValue, categoryProvenance),
			server.fact("Subcategory", subcategoryKnown, subcategoryValue, subcategoryProvenance),
		},
		Presentation: []factView{
			server.fact("Display name", presentation.DisplayName.Known, presentation.DisplayName.Value, presentation.DisplayName.Provenance),
			server.fact("Canonical name", presentation.CanonicalName.Known, presentation.CanonicalName.Value, presentation.CanonicalName.Provenance),
			server.fact("Caption", presentation.Caption.Known, presentation.Caption.Value, presentation.Caption.Provenance),
			server.fact("Description", presentation.Description.Known, presentation.Description.Value, presentation.Description.Provenance),
			server.fact("Location", presentation.Location.Known, presentation.Location.Value, presentation.Location.Provenance),
			server.fact("Icon path", presentation.IconPath.Known, presentation.IconPath.Value, presentation.IconPath.Provenance),
		},
		Storage: server.storageFacts(storage),
		Safety: []factView{
			server.fact("Cut content", safety.CutContent.Known, safety.CutContent.Value, safety.CutContent.Provenance),
			server.fact("Ban risk", safety.BanRisk.Known, safety.BanRisk.Value, safety.BanRisk.Provenance),
			server.fact("DLC", safety.DLC.Known, safety.DLC.Value, safety.DLC.Provenance),
			server.fact("No database", safety.NoDatabase.Known, safety.NoDatabase.Value, safety.NoDatabase.Provenance),
			server.fact("Scales with NG", safety.ScalesWithNG.Known, safety.ScalesWithNG.Value, safety.ScalesWithNG.Provenance),
		},
		TextMetadata:   server.readableFacts(presentation.TextMetadata, ""),
		Acquisition:    server.readableFacts(acquisition, ""),
		Modifiers:      server.readableFacts(modifiers, ""),
		Links:          server.readableFacts(links, ""),
		Unlocks:        server.readableFacts(unlocks, "Unlock"),
		TechnicalData:  server.readableFacts(technicalData, "Technical record"),
		Capabilities:   server.capabilityViewsFor(capabilities),
		FamilyData:     familyData,
		Variants:       server.variantViews(item),
		Aliases:        server.aliasViews(item),
		SourceRecords:  sourceRecords,
		Relations:      server.relationViews(view),
		UnknownCount:   unknownCount,
		HasDescription: presentation.Description.Known,
	}
}

func findVariant(item *schema.ItemDocument, gameID uint32) (schema.ItemVariant, bool) {
	if item == nil {
		return schema.ItemVariant{}, false
	}
	for _, variant := range item.Variants {
		if variant.GameID.Known && variant.GameID.Value == gameID {
			return variant, true
		}
	}
	return schema.ItemVariant{}, false
}
