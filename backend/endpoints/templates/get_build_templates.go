/*
Endpoint: GetBuildTemplates
EndpointID: get_build_templates
Purpose: Returns the Build Templates library without loading the complete contents of every template.
How it works: The runtime handler reads the metadata index of the local templates library via the templates store, applies search, tags, and pagination filters, and returns a typed list of template metadata without opening template payload files and without modifying any state.
Supported resource types: GameResource references.
Input variables: search, tags, page, pageSize.
GameCatalog variables read: none.
Save variables read: none; the getter is non-mutating and reads only the local templates library.
Implementation status: implemented; GetBuildTemplates is the runtime handler of this contract.
*/
package templates

import (
	"errors"
	"fmt"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// GetBuildTemplatesEndpointID is the stable backend identifier of GetBuildTemplates.
const GetBuildTemplatesEndpointID = "get_build_templates"

// GetBuildTemplatesDefaultPageSize is the default page size used when pageSize is 0.
const GetBuildTemplatesDefaultPageSize = 50

// GetBuildTemplatesDefinition describes the public getter contract.
var GetBuildTemplatesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetBuildTemplates",
	ID:                         GetBuildTemplatesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"search", "tags", "page", "pageSize"},
	Description:                "Returns the Build Templates library without loading the complete contents of every template.",
})

// BuildTemplateEntry is the lightweight metadata of one build template in the
// library index.
type BuildTemplateEntry = buildtemplates.TemplateMetadata

// GetBuildTemplatesResult is the typed result of GetBuildTemplates. Total counts
// every template that passed the search and tag filters, before paging.
type GetBuildTemplatesResult struct {
	Templates []BuildTemplateEntry `json:"templates"`
	Total     int                  `json:"total"`
	Page      int                  `json:"page"`
	PageSize  int                  `json:"pageSize"`
}

// GetBuildTemplates returns one page of Build Template metadata from the local
// template library store.
//
// An empty search does not filter. A non-empty search is a case-insensitive
// substring match on the template name and description without trimming or
// altering input.
//
// An empty or nil tags slice does not filter. Multiple tags have AND semantics;
// every tag must match case-sensitively and exactly on the template's tag list.
//
// Results keep the canonical legacy library order: UpdatedAt descending, with
// a deterministic tie-break on TemplateID ascending.
func GetBuildTemplates(
	store *buildtemplates.Store,
	search string,
	tags []string,
	page int,
	pageSize int,
) (GetBuildTemplatesResult, error) {
	if store == nil {
		return GetBuildTemplatesResult{}, errors.New("templates store is not available")
	}
	if page < 0 {
		return GetBuildTemplatesResult{}, fmt.Errorf("page must not be negative; got %d", page)
	}
	if pageSize < 0 {
		return GetBuildTemplatesResult{}, fmt.Errorf("pageSize must not be negative; got %d", pageSize)
	}
	for index, tag := range tags {
		if tag == "" {
			return GetBuildTemplatesResult{}, fmt.Errorf("tag %d must not be empty", index)
		}
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = GetBuildTemplatesDefaultPageSize
	}

	entries, err := store.ListTemplates()
	if err != nil {
		return GetBuildTemplatesResult{}, err
	}

	loweredSearch := strings.ToLower(search)
	matches := make([]BuildTemplateEntry, 0, len(entries))
	for _, entry := range entries {
		if search != "" &&
			!strings.Contains(strings.ToLower(entry.Name), loweredSearch) &&
			!strings.Contains(strings.ToLower(entry.Description), loweredSearch) {
			continue
		}

		if len(tags) > 0 {
			matchesAll := true
			for _, wanted := range tags {
				found := false
				for _, tag := range entry.Tags {
					if tag == wanted {
						found = true
						break
					}
				}
				if !found {
					matchesAll = false
					break
				}
			}
			if !matchesAll {
				continue
			}
		}

		matches = append(matches, entry)
	}

	total := len(matches)
	if total == 0 || page-1 > (total-1)/pageSize {
		return GetBuildTemplatesResult{
			Templates: []BuildTemplateEntry{},
			Total:     total,
			Page:      page,
			PageSize:  pageSize,
		}, nil
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	return GetBuildTemplatesResult{
		Templates: matches[start:end],
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}, nil
}
