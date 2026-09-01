package contract

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// This test is the single guard against the two dictionaries drifting apart:
// SaveForge 2.0 owns its stable mutation kinds in SaveEngine, and every
// implemented save-session mutation endpoint must use its own EndpointID as
// exactly one of them.
//
// A contract.Mutation is not automatically a save-session mutation. Three
// classes exist and only the first one commits a save revision:
//
//   - save-session mutations, which advance the session's saveRevision;
//   - application and store mutations, which change the session registry or the
//     local template store and never advance a revision;
//   - contract-only definitions, which have no runtime handler at all.

// notSaveSessionMutations lists every implemented contract.Mutation that does
// not commit a save-session revision, with the surface it changes instead. A
// receipt must never demand a saveSessionID from one of these.
var notSaveSessionMutations = map[string]string{
	"load_save":             "creates a session; there is no earlier revision to advance",
	"close_save":            "removes a session from the engine registry",
	"create_build_template": "writes the local build-template store",
	"update_build_template": "writes the local build-template store",
	"delete_build_template": "writes the local build-template store",
	"set_diagnostic_mode":   "contract definition only; it has no runtime handler",
}

var mutationKindPattern = regexp.MustCompile(`Kind:\s*contract\.Mutation\b`)
var endpointIDPattern = regexp.MustCompile(`EndpointID = ("[a-z0-9_]+")`)
var implementationStatusPattern = regexp.MustCompile(`Implementation status:\s*(.+)`)

func TestSaveSessionMutationEndpointIDsMatchSaveEngineMutationKinds(t *testing.T) {
	t.Parallel()

	saveSession := make(map[string]string)
	for id, path := range mutationEndpointIDs(t) {
		if _, excluded := notSaveSessionMutations[id]; excluded {
			continue
		}
		saveSession[id] = path
	}

	registered := make(map[string]bool, len(saveengine.MutationKinds()))
	for _, kind := range saveengine.MutationKinds() {
		registered[kind] = true
	}

	for id, path := range saveSession {
		if !registered[id] {
			t.Errorf("%s is an implemented save-session mutation but SaveEngine registers no operation kind %q",
				path, id)
		}
	}
	for kind := range registered {
		if _, exists := saveSession[kind]; !exists {
			t.Errorf("SaveEngine registers operation kind %q, which is no implemented save-session mutation EndpointID",
				kind)
		}
	}
}

// Every registered kind must resolve to a deterministic, duplicate-free and
// non-empty scope list. The scope contract is public, so an unresolvable kind
// is a defect at this boundary, not only inside SaveEngine.
func TestEveryMutationKindResolvesExactChangedScopes(t *testing.T) {
	t.Parallel()

	for _, kind := range saveengine.MutationKinds() {
		scopes, err := saveengine.ChangedScopesForMutationKind(kind)
		if err != nil {
			t.Fatalf("ChangedScopesForMutationKind(%q): %v", kind, err)
		}
		if len(scopes) == 0 {
			t.Errorf("operation kind %q resolves to no changed scope", kind)
		}
		seen := make(map[string]bool, len(scopes))
		for _, scope := range scopes {
			if scope == "" {
				t.Errorf("operation kind %q resolves to an empty scope", kind)
			}
			if seen[scope] {
				t.Errorf("operation kind %q repeats scope %q", kind, scope)
			}
			seen[scope] = true
		}

		again, err := saveengine.ChangedScopesForMutationKind(kind)
		if err != nil {
			t.Fatalf("ChangedScopesForMutationKind(%q) repeated: %v", kind, err)
		}
		if strings.Join(scopes, ",") != strings.Join(again, ",") {
			t.Errorf("operation kind %q resolved %v and then %v, want one deterministic order",
				kind, scopes, again)
		}
	}
}

// Every excluded endpoint must still exist as an implemented or contract-only
// mutation, so the classification cannot silently keep a stale entry.
func TestNotSaveSessionMutationsAreRealMutationEndpoints(t *testing.T) {
	t.Parallel()

	known := mutationEndpointIDs(t)
	for id := range notSaveSessionMutations {
		if _, exists := known[id]; !exists {
			t.Errorf("%q is classified as a non-save-session mutation but no mutation endpoint declares it", id)
		}
	}
}

// mutationEndpointIDs returns the EndpointID of every contract.Mutation file,
// mapped to its path. Contract-only files are included: their classification is
// checked, only their absence from the kind registry is expected.
func mutationEndpointIDs(t *testing.T) map[string]string {
	t.Helper()

	entries, err := os.ReadDir(endpointsDir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", endpointsDir, err)
	}

	found := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "contract" || entry.Name() == "itemrouting" ||
			entry.Name() == "swagger" {
			continue
		}
		packageDir := filepath.Join(endpointsDir, entry.Name())
		files, err := os.ReadDir(packageDir)
		if err != nil {
			t.Fatalf("ReadDir(%s): %v", packageDir, err)
		}
		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".go" ||
				strings.HasSuffix(file.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(packageDir, file.Name())
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", path, err)
			}
			if !mutationKindPattern.Match(source) {
				continue
			}
			match := endpointIDPattern.FindSubmatch(source)
			if match == nil {
				t.Fatalf("%s declares contract.Mutation without an EndpointID constant", path)
			}
			id, err := strconv.Unquote(string(match[1]))
			if err != nil {
				t.Fatalf("%s EndpointID: %v", path, err)
			}
			status := implementationStatusPattern.FindSubmatch(source)
			if status == nil {
				t.Fatalf("%s declares no implementation status", path)
			}
			found[id] = path
		}
	}
	return found
}
