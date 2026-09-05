package application

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/hostsettings"
)

// TestHostSettingsRoundTrip covers the getter and the setter together: the
// closed policy vocabulary is reported, a stored value comes back, and an
// unknown policy is refused rather than mapped onto the default.
func TestHostSettingsRoundTrip(t *testing.T) {
	store := hostsettings.NewStore(t.TempDir())

	initial, err := GetHostSettings(store, nil)
	if err != nil {
		t.Fatalf("GetHostSettings: %v", err)
	}
	if initial.RemoteBackupPolicy != "ask" || initial.DefaultRemoteBackupPolicy != "ask" {
		t.Fatalf("initial = %+v, want the ask policy", initial)
	}
	if len(initial.AvailableRemoteBackupPolicies) != 2 {
		t.Fatalf("available policies = %v, want exactly ask and always",
			initial.AvailableRemoteBackupPolicies)
	}
	if !initial.ConfigurationDirectoryExists || initial.LogDirectoryExists {
		t.Fatalf("a store with a state directory reports %+v", initial)
	}

	stored, err := SetHostSettings(store, nil, true, "always")
	if err != nil {
		t.Fatalf("SetHostSettings: %v", err)
	}
	if !stored.SkipReviewForNormalRisk || stored.RemoteBackupPolicy != "always" {
		t.Fatalf("stored = %+v", stored)
	}
	if _, err := SetHostSettings(store, nil, false, "never"); err == nil {
		t.Fatal("SetHostSettings accepted a policy that would disable the mandatory backup")
	}
	// The refused write left the stored value alone.
	after, err := GetHostSettings(store, nil)
	if err != nil {
		t.Fatalf("GetHostSettings after the refusal: %v", err)
	}
	if after.RemoteBackupPolicy != "always" {
		t.Fatalf("after = %+v, want the previously stored policy", after)
	}
}

// TestExportDiagnosticReportWritesOnlyRedactedData is the security contract of
// the export: whatever else the report gains later, it may never carry a save
// path, a key path or save bytes.
func TestExportDiagnosticReportWritesOnlyRedactedData(t *testing.T) {
	store := hostsettings.NewStore(t.TempDir())
	if _, err := store.Set(false, "always"); err != nil {
		t.Fatalf("Set host settings: %v", err)
	}
	target := filepath.Join(t.TempDir(), "report.json")

	result, err := ExportDiagnosticReport("2.0.0", store, nil, nil, "", target)
	if err != nil {
		t.Fatalf("ExportDiagnosticReport: %v", err)
	}
	if !result.Exported || result.RecordCount != 0 {
		t.Fatalf("recordCount = %d, want none without a session", result.RecordCount)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read the report: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("the report is not valid JSON: %v", err)
	}
	if document["applicationVersion"] != "2.0.0" {
		t.Fatalf("applicationVersion = %v", document["applicationVersion"])
	}
	settings, ok := document["settings"].(map[string]any)
	if !ok || settings["remoteBackupPolicy"] != "always" {
		t.Fatalf("settings = %v", document["settings"])
	}
	if _, present := document["session"]; present {
		t.Fatal("the report describes a session although none was named")
	}
	// The document carries no field that could hold a host path or key
	// material, and no such value is anywhere in the bytes.
	for _, forbidden := range []string{"savePath", "sourcePath", "keyPath", "configurationDirectory"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("the report carries a %q field", forbidden)
		}
	}
	if strings.Contains(string(data), store.Directory()) {
		t.Fatal("the report carries the host state directory")
	}

	if _, err := ExportDiagnosticReport("2.0.0", store, nil, nil, "", ""); err == nil {
		t.Fatal("ExportDiagnosticReport accepted an empty target")
	}
}

// TestProjectLinksAreAClosedAllowlist: the frontend can only ever ask for one
// of these identifiers, and an unknown one produces no address at all.
func TestProjectLinksAreAClosedAllowlist(t *testing.T) {
	result, err := GetProjectLinks()
	if err != nil {
		t.Fatalf("GetProjectLinks: %v", err)
	}
	if len(result.Links) != 4 {
		t.Fatalf("links = %+v, want the four approved destinations", result.Links)
	}
	for _, link := range result.Links {
		if !strings.HasPrefix(link.URL, "https://") {
			t.Fatalf("link %q is not an https address: %q", link.ID, link.URL)
		}
		resolved, err := ResolveProjectLink(link.ID)
		if err != nil || resolved != link.URL {
			t.Fatalf("ResolveProjectLink(%q) = %q, %v", link.ID, resolved, err)
		}
	}
	if _, err := ResolveProjectLink("https://example.com"); err == nil {
		t.Fatal("ResolveProjectLink accepted an address instead of an identifier")
	}
	if _, err := ResolveProjectLink("unknown"); err == nil {
		t.Fatal("ResolveProjectLink accepted an unknown identifier")
	}
}

// TestCheckForUpdatesIgnoresDraftsAndPrereleases runs against a local mock
// server only. It covers the whole decision: which releases count, which
// version wins, and what an uncomparable running version reports.
func TestCheckForUpdatesIgnoresDraftsAndPrereleases(t *testing.T) {
	var sentAuthorization, sentCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sentAuthorization = r.Header.Get("Authorization")
		sentCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"tag_name":"v9.9.9","draft":true,"prerelease":false},
			{"tag_name":"v3.0.0-rc1","draft":false,"prerelease":true},
			{"tag_name":"v2.1.0","draft":false,"prerelease":false,"published_at":"2026-02-01T00:00:00Z"},
			{"tag_name":"v2.0.0","draft":false,"prerelease":false},
			{"tag_name":"nightly","draft":false,"prerelease":false}
		]`))
	}))
	defer server.Close()

	available, err := CheckForUpdates(context.Background(), "2.0.0", server.URL)
	if err != nil {
		t.Fatalf("CheckForUpdates: %v", err)
	}
	if available.Status != UpdateStatusAvailable || available.LatestVersion != "v2.1.0" {
		t.Fatalf("result = %+v, want v2.1.0 reported as available", available)
	}
	if !available.ComparisonPossible || available.PublishedAt != "2026-02-01T00:00:00Z" {
		t.Fatalf("result = %+v", available)
	}
	if sentAuthorization != "" || sentCookie != "" {
		t.Fatalf("the check sent credentials: authorization=%q cookie=%q",
			sentAuthorization, sentCookie)
	}

	current, err := CheckForUpdates(context.Background(), "2.1.0", server.URL)
	if err != nil {
		t.Fatalf("CheckForUpdates: %v", err)
	}
	if current.Status != UpdateStatusCurrent {
		t.Fatalf("result = %+v, want the running version reported as current", current)
	}

	// A development build has no comparable version, and saying so is the only
	// honest answer.
	development, err := CheckForUpdates(context.Background(), "dev", server.URL)
	if err != nil {
		t.Fatalf("CheckForUpdates: %v", err)
	}
	if development.Status != UpdateStatusUnknown || development.ComparisonPossible {
		t.Fatalf("result = %+v, want an uncomparable running version", development)
	}
}

// TestCheckForUpdatesReportsAFailureSafely: an upstream fault becomes a stable
// backend failure, never an application-shaped sentence built from the
// service's own answer.
func TestCheckForUpdatesReportsAFailureSafely(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded for 203.0.113.7"}`))
	}))
	defer server.Close()

	_, err := CheckForUpdates(context.Background(), "2.0.0", server.URL)
	if err == nil {
		t.Fatal("CheckForUpdates reported success for a refused request")
	}
	if strings.Contains(err.Error(), "203.0.113.7") || strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("the failure repeats the upstream answer: %v", err)
	}

	oversized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", updateCheckResponseLimit+1)))
	}))
	defer oversized.Close()
	if _, err := CheckForUpdates(context.Background(), "2.0.0", oversized.URL); err == nil {
		t.Fatal("CheckForUpdates accepted an oversized answer")
	}

	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer redirectTarget.Close()
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL, http.StatusFound)
	}))
	defer redirecting.Close()
	if _, err := CheckForUpdates(context.Background(), "2.0.0", redirecting.URL); err == nil {
		t.Fatal("CheckForUpdates followed a redirect outside the approved origin")
	}
}

// TestCheckForUpdatesReportsNoStableRelease covers the empty library, which is
// a truthful "nothing published" rather than an error.
func TestCheckForUpdatesReportsNoStableRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"tag_name":"v1.0.0-beta","prerelease":true}]`))
	}))
	defer server.Close()

	result, err := CheckForUpdates(context.Background(), "2.0.0", server.URL)
	if err != nil {
		t.Fatalf("CheckForUpdates: %v", err)
	}
	if result.Status != UpdateStatusUnavailable || result.LatestVersion != "" {
		t.Fatalf("result = %+v, want no stable release", result)
	}
}

// The report gained the diagnostic mode state and a bounded slice of the
// instance-wide records. Both go through the same sanitisation boundary as the
// console, so an attempt to record a private value must leave no trace in the
// exported document either.
func TestExportDiagnosticReportCarriesTheModeAndSafeEventsOnly(t *testing.T) {
	store := hostsettings.NewStore(t.TempDir())
	service := diagnostics.NewService(diagnostics.Options{Directory: t.TempDir()})
	t.Cleanup(service.Close)

	service.SetDebugMode(true)
	service.Log(diagnostics.Entry{
		Event: diagnostics.EventOperationFinished, Operation: diagnostics.OperationDeployToTarget,
		Status: diagnostics.StatusSucceeded, CorrelationID: "0123456789abcdef",
	})
	// None of these is part of the closed catalogue, so none of them may reach
	// the document.
	private := []string{
		"/Users/oisis/Documents/ER0000.sl2",
		"deck@192.168.0.10",
		"76561198000000000",
		"~/.ssh/id_ed25519",
	}
	for _, value := range private {
		service.Log(diagnostics.Entry{Event: value})
		service.Log(diagnostics.Entry{Event: diagnostics.EventApplicationStarted, Version: value})
		service.Log(diagnostics.Entry{Event: diagnostics.EventOperationStarted, CorrelationID: value})
		service.Log(diagnostics.Entry{
			Event:  diagnostics.EventOperationFinished,
			Status: diagnostics.StatusFailed, Code: value,
		})
	}

	target := filepath.Join(t.TempDir(), "report.json")
	result, err := ExportDiagnosticReport("2.0.0", store, service, nil, "", target)
	if err != nil {
		t.Fatalf("ExportDiagnosticReport: %v", err)
	}
	// The mode change and the one accepted operation record: nothing that was
	// rejected was counted or carried.
	if result.EventCount != 2 {
		t.Fatalf("eventCount = %d, want 2", result.EventCount)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read the report: %v", err)
	}
	for _, value := range private {
		if strings.Contains(string(data), value) {
			t.Errorf("the report carries the private value %q", value)
		}
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("the report is not valid JSON: %v", err)
	}
	mode, ok := document["diagnostics"].(map[string]any)
	if !ok || mode["enabled"] != true {
		t.Fatalf("diagnostics = %v, want the enabled mode state", document["diagnostics"])
	}
	events, ok := document["events"].([]any)
	if !ok || len(events) != 2 {
		t.Fatalf("events = %v, want the two accepted records", document["events"])
	}
}
