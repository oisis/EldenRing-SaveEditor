/*
Endpoint: ImportBuildTemplate
EndpointID: import_build_template
Purpose: Adds a Build Template document the user picked in the native file dialog to the local templates library.
How it works: The runtime handler reads the chosen local file under a size bound, decodes and validates it against the Build Template schema and stores it as a new template. It contacts no network and reads no save.
Supported resource types: GameResource references.
Input variables: source.
GameCatalog variables read: none.
Save variables read: none; the import only writes into the local templates library.
Implementation status: implemented
*/
package templates

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// ImportBuildTemplateEndpointID is the stable backend identifier of
// ImportBuildTemplate.
const ImportBuildTemplateEndpointID = "import_build_template"

// importSizeLimit bounds how much of a chosen document is read at all, so a
// mistakenly picked large file cannot be loaded into memory whole.
const importSizeLimit = 8 << 20

// ImportBuildTemplateDefinition describes the public mutation contract.
var ImportBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ImportBuildTemplate",
	ID:                         ImportBuildTemplateEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"source"},
	Description:                "Adds a Build Template document the user picked in the native file dialog to the local templates library.",
})

// ImportBuildTemplateResult is the typed result of ImportBuildTemplate.
type ImportBuildTemplateResult struct {
	TemplateID       string `json:"templateID"`
	TemplateRevision string `json:"templateRevision"`
}

// ImportBuildTemplate stores one validated template document.
//
// source is the path the host's own dialog returned. A cancelled dialog never
// reaches this endpoint: the bridge treats the empty path as an ordinary
// outcome. The import is local only — there is deliberately no way to state a
// URL, and nothing here performs a network request.
func ImportBuildTemplate(
	store *buildtemplates.Store, source string,
) (ImportBuildTemplateResult, error) {
	if store == nil {
		return ImportBuildTemplateResult{}, errors.New("templates store is not available")
	}
	if source == "" {
		return ImportBuildTemplateResult{}, errors.New("a template import needs a source document")
	}
	file, err := os.Open(source)
	if err != nil {
		// The host path is not repeated back: it is the user's own private
		// location and adds nothing the interface can act on.
		return ImportBuildTemplateResult{}, errors.New("the selected template document could not be read")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return ImportBuildTemplateResult{}, errors.New("the selected template document could not be read")
	}
	if !info.Mode().IsRegular() {
		return ImportBuildTemplateResult{}, errors.New("the selected template document is not a regular file")
	}
	if info.Size() > importSizeLimit {
		return ImportBuildTemplateResult{}, fmt.Errorf(
			"a Build Template document must not exceed %d bytes", importSizeLimit)
	}
	data, err := io.ReadAll(io.LimitReader(file, importSizeLimit+1))
	if err != nil {
		return ImportBuildTemplateResult{}, errors.New("the selected template document could not be read")
	}
	// DecodeTemplate is the library's own validation. The import adds no second,
	// weaker check of its own and rejects anything it refuses.
	template, err := buildtemplates.DecodeTemplate(data)
	if err != nil {
		return ImportBuildTemplateResult{}, fmt.Errorf("the selected document is not a valid Build Template: %w", err)
	}
	templateID, templateRevision, err := store.CreateTemplate(template)
	if err != nil {
		return ImportBuildTemplateResult{}, err
	}
	return ImportBuildTemplateResult{TemplateID: templateID, TemplateRevision: templateRevision}, nil
}
