package dbviewer

import (
	"net/http"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const catalogAssetURLPrefix = "/catalog-assets/"

func (server *Server) iconAssetHandler(response http.ResponseWriter, request *http.Request) {
	assetPath := "assets/" + request.PathValue("assetPath")
	content, mediaType, exists := server.data.ReadAssetWithMediaType(assetPath)
	if !exists {
		http.NotFound(response, request)
		return
	}
	response.Header().Set("Content-Type", mediaType)
	_, _ = response.Write(content)
}

func itemIconURL(item *schema.ItemDocument) string {
	if item == nil || !item.Presentation.IconPath.Known {
		return ""
	}
	return iconURL(item.Presentation.IconPath.Value)
}

func variantIconURL(item *schema.ItemDocument, variant schema.ItemVariant) string {
	if variant.Data.Presentation.IconPath.Known {
		return iconURL(variant.Data.Presentation.IconPath.Value)
	}
	return itemIconURL(item)
}

func iconURL(iconPath string) string {
	return catalogAssetURLPrefix + strings.TrimPrefix(iconPath, "assets/")
}
