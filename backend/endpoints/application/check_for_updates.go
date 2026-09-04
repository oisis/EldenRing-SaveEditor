/*
Endpoint: CheckForUpdates
EndpointID: check_for_updates
Purpose: Checks the project's official GitHub Releases once, on an explicit user action, and reports whether a newer stable release exists.
How it works: The runtime handler performs one bounded HTTPS GET against the project's releases API, discards drafts and prereleases, compares the newest remaining tag with the running application version and returns a typed answer. It downloads and installs nothing, sends no credentials and keeps no background schedule.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables read: none.
Implementation status: implemented
*/
package application

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// CheckForUpdatesEndpointID is the stable backend identifier of CheckForUpdates.
const CheckForUpdatesEndpointID = "check_for_updates"

// CheckForUpdatesDefinition describes the public getter contract.
var CheckForUpdatesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "CheckForUpdates",
	ID:                         CheckForUpdatesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: nil,
	Description:                "Checks the project's official GitHub Releases once, on an explicit user action, and reports whether a newer stable release exists.",
})

// releasesEndpoint is the only address this endpoint ever contacts. It is a
// compile-time constant: no setting, no save and no frontend argument can point
// the update check at a different host.
const releasesEndpoint = "https://api.github.com/repos/oisis/EldenRing-SaveEditor/releases?per_page=30"

// updateCheckTimeout bounds the whole request, including connection setup and
// the body read.
const updateCheckTimeout = 10 * time.Second

// updateCheckResponseLimit bounds how much of the answer is read at all, so a
// hostile or broken responder cannot make the check consume memory without end.
const updateCheckResponseLimit = 1 << 20

// The three outcomes the screen renders. They are stable codes rather than
// sentences: the frontend owns the wording and its localisation.
const (
	UpdateStatusCurrent     = "current"
	UpdateStatusAvailable   = "available"
	UpdateStatusUnknown     = "unknown"
	UpdateStatusUnavailable = "unavailable"
)

// CheckForUpdatesResult is the typed result of CheckForUpdates.
//
// LatestVersion carries the newest stable tag exactly as GitHub published it and
// is empty when no stable release exists or the check failed. ReleaseURL is only
// ever the project's own release page.
type CheckForUpdatesResult struct {
	Status             string `json:"status"`
	CurrentVersion     string `json:"currentVersion"`
	LatestVersion      string `json:"latestVersion,omitempty"`
	ReleaseURL         string `json:"releaseURL,omitempty"`
	PublishedAt        string `json:"publishedAt,omitempty"`
	ComparisonPossible bool   `json:"comparisonPossible"`
}

// githubRelease is the subset of one release entry this endpoint reads. Every
// other field GitHub sends is ignored rather than carried further.
type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
}

// CheckForUpdates performs the single manual check.
//
// endpointOverride exists for a test against a local mock server and is empty in
// production; the frontend cannot reach it, because the bridge never forwards a
// value for it.
func CheckForUpdates(
	ctx context.Context,
	applicationVersion string,
	endpointOverride string,
) (CheckForUpdatesResult, error) {
	if applicationVersion == "" {
		return CheckForUpdatesResult{}, errors.New("application version is required")
	}
	address := releasesEndpoint
	if endpointOverride != "" {
		address = endpointOverride
	}

	result := CheckForUpdatesResult{
		Status:         UpdateStatusUnknown,
		CurrentVersion: applicationVersion,
		ReleaseURL:     projectLinks[ProjectLinkReleases],
	}

	requestContext, cancel := context.WithTimeout(ctx, updateCheckTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, address, nil)
	if err != nil {
		return CheckForUpdatesResult{}, err
	}
	// Only the two headers the public API needs. No token, no cookie and no
	// user identity is ever attached: the check is anonymous by construction.
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "SaveForge")

	// A dedicated client rather than http.DefaultClient, so no other part of the
	// process can install a cookie jar or redirect policy this call would
	// inherit. The standard transport may still honor the host's proxy settings.
	origin, err := url.Parse(address)
	if err != nil {
		return CheckForUpdatesResult{}, errors.New("the update service address is invalid")
	}
	client := &http.Client{
		Timeout: updateCheckTimeout,
		CheckRedirect: func(request *http.Request, _ []*http.Request) error {
			if request.URL.Scheme != origin.Scheme || request.URL.Host != origin.Host {
				return errors.New("the update service redirected outside its approved origin")
			}
			return nil
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return CheckForUpdatesResult{}, errors.New("the update check could not reach the release service")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		// The upstream status is not repeated to the user: it is not actionable
		// and can carry rate-limit detail that reads as an application fault.
		return CheckForUpdatesResult{}, errors.New("the release service did not answer the update check")
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, updateCheckResponseLimit+1))
	if err != nil {
		return CheckForUpdatesResult{}, errors.New("the update check answer could not be read")
	}
	if len(body) > updateCheckResponseLimit {
		return CheckForUpdatesResult{}, errors.New("the update check answer was too large")
	}
	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return CheckForUpdatesResult{}, errors.New("the update check answer could not be understood")
	}

	newest := githubRelease{}
	newestVersion := []int(nil)
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}
		version, ok := parseReleaseVersion(release.TagName)
		if !ok {
			continue
		}
		if newestVersion == nil || compareVersions(version, newestVersion) > 0 {
			newest = release
			newestVersion = version
		}
	}
	if newestVersion == nil {
		result.Status = UpdateStatusUnavailable
		return result, nil
	}

	result.LatestVersion = newest.TagName
	result.PublishedAt = newest.PublishedAt
	current, ok := parseReleaseVersion(applicationVersion)
	if !ok {
		// A development build carries no comparable version. Saying so is
		// honest; guessing "up to date" would not be.
		result.Status = UpdateStatusUnknown
		return result, nil
	}
	result.ComparisonPossible = true
	if compareVersions(newestVersion, current) > 0 {
		result.Status = UpdateStatusAvailable
	} else {
		result.Status = UpdateStatusCurrent
	}
	return result, nil
}

// parseReleaseVersion reads a "1.2.3" or "v1.2.3" tag. Anything else — a tag
// with a suffix, a date tag, a development version — is reported as
// uncomparable rather than coerced into a number.
func parseReleaseVersion(tag string) ([]int, bool) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(tag), "v")
	if trimmed == "" {
		return nil, false
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) != 3 {
		return nil, false
	}
	version := make([]int, 0, 3)
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return nil, false
		}
		version = append(version, value)
	}
	return version, true
}

func compareVersions(left, right []int) int {
	for index := range left {
		if left[index] != right[index] {
			if left[index] > right[index] {
				return 1
			}
			return -1
		}
	}
	return 0
}
