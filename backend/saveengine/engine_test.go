package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// pcFixtureSize and ps4FixtureSize are the smallest sizes the first stage
// accepts for each container.
const (
	pcFixtureSize  = int64(pcHeaderSize) + pcSlotsSize + pcUserData10BlockSize
	ps4FixtureSize = int64(ps4HeaderSize) + ps4SlotsSize + ps4UserData10Size
)

// writeFixture writes header at the start of a synthetic save file and grows it
// to size. The body stays sparse: the stage validates bounds, not content.
func writeFixture(t *testing.T, name string, header []byte, size int64) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	defer file.Close()
	if _, err := file.Write(header); err != nil {
		t.Fatalf("write fixture header: %v", err)
	}
	if err := file.Truncate(size); err != nil {
		t.Fatalf("size fixture: %v", err)
	}
	return path
}

func pcHeader() []byte {
	header := make([]byte, pcHeaderSize)
	copy(header, pcMagic)
	binary.LittleEndian.PutUint32(header[pcEntryCountOffset:], pcEntryCount)
	return header
}

func ps4Header() []byte {
	header := make([]byte, ps4HeaderSize)
	copy(header, ps4Magic)
	for entry := 0; entry < ps4EntryCount; entry++ {
		offset := ps4EntryTableOffset + entry*ps4EntryStride
		binary.LittleEndian.PutUint32(header[offset:], uint32(ps4FirstEntryIndex+entry))
		binary.LittleEndian.PutUint32(header[offset+4:], ps4EntryMarker)
	}
	return header
}

func TestLoadSaveCreatesSessionForPCContainer(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)

	info, err := New().LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if info.SaveSessionID == "" {
		t.Error("saveSessionID is empty")
	}
	if info.Platform != string(PlatformPC) {
		t.Errorf("platform = %q, want %q", info.Platform, PlatformPC)
	}
	if info.Format != pcContainerFormat {
		t.Errorf("format = %q, want %q", info.Format, pcContainerFormat)
	}
	if info.UnsavedChanges {
		t.Error("a freshly loaded session reports unsaved changes")
	}
}

func TestLoadSaveCreatesSessionForPS4Container(t *testing.T) {
	path := writeFixture(t, "ps4.sl2", ps4Header(), ps4FixtureSize)

	info, err := New().LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if info.SaveSessionID == "" {
		t.Error("saveSessionID is empty")
	}
	if info.Platform != string(PlatformPS4) {
		t.Errorf("platform = %q, want %q", info.Platform, PlatformPS4)
	}
	if info.Format != ps4ContainerFormat {
		t.Errorf("format = %q, want %q", info.Format, ps4ContainerFormat)
	}
	if info.UnsavedChanges {
		t.Error("a freshly loaded session reports unsaved changes")
	}
}

func TestLoadSaveGivesEverySessionItsOwnIdentifier(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	first, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("first LoadSave: %v", err)
	}
	second, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("second LoadSave: %v", err)
	}
	if first.SaveSessionID == second.SaveSessionID {
		t.Errorf("two sessions share the identifier %q", first.SaveSessionID)
	}
	if len(engine.sessions) != 2 {
		t.Errorf("engine holds %d sessions, want 2", len(engine.sessions))
	}
}

func TestLoadSaveKeepsPrivateSnapshotOfTheSource(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	info, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// The source changes after a successful load. The session keeps the data it
	// was created from.
	if err := os.WriteFile(path, []byte("NOPE not a save"), 0o600); err != nil {
		t.Fatalf("overwrite source: %v", err)
	}

	loaded := engine.sessions[info.SaveSessionID]
	if loaded == nil {
		t.Fatalf("engine holds no session for %q", info.SaveSessionID)
	}
	if !bytes.HasPrefix(loaded.snapshot.data, pcMagic) {
		t.Errorf("snapshot starts with %q, want the original magic %q",
			loaded.snapshot.data[:min(len(loaded.snapshot.data), len(pcMagic))], pcMagic)
	}
	if loaded.snapshot.length() != pcFixtureSize {
		t.Errorf("snapshot length = 0x%X, want 0x%X", loaded.snapshot.length(), pcFixtureSize)
	}

	// The recorded source metadata is what the snapshot was created from and is
	// not re-read: the path still names the file even though its content is now
	// something else entirely, and the session neither noticed nor cared.
	if info.SourcePath != path {
		t.Errorf("sourcePath = %q, want the exact load path %q", info.SourcePath, path)
	}
	after, err := engine.GetSessionInfo(info.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after the source changed: %v", err)
	}
	if after != info {
		t.Errorf("overwriting the source changed the session to %+v, want %+v", after, info)
	}

	// The public metadata still carries no source data beyond the approved
	// fields: no bytes, and no field the approved set does not name.
	if fields := reflect.TypeOf(info).NumField(); fields != len(sessionInfoFields) {
		t.Errorf("SessionInfo has %d fields, want the %d approved ones",
			fields, len(sessionInfoFields))
	}
	for name, value := range map[string]string{
		"saveSessionID": info.SaveSessionID,
		"platform":      info.Platform,
		"format":        info.Format,
		"sourceKind":    info.SourceKind,
		"saveRevision":  info.SaveRevision,
	} {
		if strings.Contains(value, path) || strings.Contains(value, string(pcMagic)) {
			t.Errorf("%s = %q exposes the source path or its content", name, value)
		}
	}
	if strings.Contains(info.SourcePath, string(pcMagic)) {
		t.Errorf("sourcePath = %q exposes the source content", info.SourcePath)
	}
}

func TestLoadSaveRejectsUnknownContainer(t *testing.T) {
	unknown := make([]byte, 0x100)
	copy(unknown, []byte("NOPE"))

	cases := map[string]string{
		"unknown magic":        writeFixture(t, "unknown.sl2", unknown, pcFixtureSize),
		"shorter than a magic": writeFixture(t, "tiny.sl2", []byte{0xCB, 0x01}, 2),
		"empty file":           writeFixture(t, "empty.sl2", nil, 0),
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := New().LoadSave(path, "", "local"); err == nil {
				t.Fatal("LoadSave accepted an unsupported container")
			}
		})
	}
}

func TestLoadSaveRejectsTruncatedContainer(t *testing.T) {
	cases := []struct {
		name   string
		header []byte
		size   int64
	}{
		{"PC missing one byte of UserData10", pcHeader(), pcFixtureSize - 1},
		{"PC header only", pcHeader(), pcHeaderSize},
		{"PC shorter than its header", pcHeader(), pcHeaderSize - 1},
		{"PS4 missing one byte of UserData10", ps4Header(), ps4FixtureSize - 1},
		{"PS4 header only", ps4Header(), ps4HeaderSize},
		{"PS4 shorter than its header", ps4Header(), ps4HeaderSize - 1},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path := writeFixture(t, "truncated.sl2", testCase.header, testCase.size)
			if _, err := New().LoadSave(path, "", "local"); err == nil {
				t.Fatal("LoadSave accepted a truncated container")
			}
		})
	}
}

func TestLoadSaveRejectsInconsistentPCEntryCount(t *testing.T) {
	header := pcHeader()
	binary.LittleEndian.PutUint32(header[pcEntryCountOffset:], pcEntryCount-1)
	path := writeFixture(t, "pc.sl2", header, pcFixtureSize)

	if _, err := New().LoadSave(path, "", "local"); err == nil {
		t.Fatal("LoadSave accepted a PC container declaring the wrong entry count")
	}
}

func TestLoadSaveRejectsInconsistentPS4Header(t *testing.T) {
	header := ps4Header()
	binary.LittleEndian.PutUint32(header[ps4EntryTableOffset:], 0)
	path := writeFixture(t, "ps4.sl2", header, ps4FixtureSize)

	if _, err := New().LoadSave(path, "", "local"); err == nil {
		t.Fatal("LoadSave accepted a PS4 container with an inconsistent header")
	}
}

func TestLoadSaveRejectsPlatformMismatch(t *testing.T) {
	cases := []struct {
		name     string
		header   []byte
		size     int64
		expected string
	}{
		{"PS4 file expected to be PC", ps4Header(), ps4FixtureSize, string(PlatformPC)},
		{"PC file expected to be PS4", pcHeader(), pcFixtureSize, string(PlatformPS4)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path := writeFixture(t, "save.sl2", testCase.header, testCase.size)
			engine := New()

			info, err := engine.LoadSave(path, testCase.expected, "local")
			if err == nil {
				t.Fatal("LoadSave accepted a save of the wrong platform")
			}
			if info.SaveSessionID != "" {
				t.Errorf("a session was created despite the mismatch: %q", info.SaveSessionID)
			}
			if len(engine.sessions) != 0 {
				t.Errorf("engine holds %d sessions after a rejected load, want 0", len(engine.sessions))
			}
		})
	}
}

func TestLoadSaveAcceptsMatchingExpectedPlatform(t *testing.T) {
	pcPath := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	if _, err := New().LoadSave(pcPath, string(PlatformPC), "local"); err != nil {
		t.Errorf("LoadSave(pc): %v", err)
	}
	ps4Path := writeFixture(t, "ps4.sl2", ps4Header(), ps4FixtureSize)
	if _, err := New().LoadSave(ps4Path, string(PlatformPS4), "local"); err != nil {
		t.Errorf("LoadSave(ps4): %v", err)
	}
}

func TestLoadSaveRejectsUnknownExpectedPlatform(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)

	for _, expected := range []string{"PC", "pc ", " pc", "ps5", "playstation"} {
		t.Run(expected, func(t *testing.T) {
			if _, err := New().LoadSave(path, expected, "local"); err == nil {
				t.Fatalf("LoadSave accepted expectedPlatform %q", expected)
			}
		})
	}
}

func TestLoadSaveRejectsDirectorySource(t *testing.T) {
	if _, err := New().LoadSave(t.TempDir(), "", "local"); err == nil {
		t.Fatal("LoadSave accepted a directory as a save source")
	}
}

func TestGetSessionInfoReturnsTheMetadataOfALoadedSession(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info != loaded {
		t.Errorf("GetSessionInfo = %+v, want the metadata LoadSave returned %+v", info, loaded)
	}
}

func TestGetSessionInfoRejectsEmptyIdentifier(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	if _, err := engine.LoadSave(path, "", "local"); err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	info, err := engine.GetSessionInfo("")
	if err == nil {
		t.Fatal("GetSessionInfo accepted an empty saveSessionID")
	}
	if info != (SessionInfo{}) {
		t.Errorf("GetSessionInfo returned %+v for an empty identifier, want the zero value", info)
	}
}

func TestGetSessionInfoRejectsUnknownIdentifier(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// Neither a foreign identifier nor a near miss of the real one resolves: the
	// identifier is matched exactly and never trimmed or normalised.
	unknown := []string{
		"00000000000000000000000000000000",
		" " + loaded.SaveSessionID,
		loaded.SaveSessionID + " ",
		strings.ToUpper(loaded.SaveSessionID),
	}
	for _, saveSessionID := range unknown {
		t.Run(saveSessionID, func(t *testing.T) {
			if _, err := engine.GetSessionInfo(saveSessionID); err == nil {
				t.Fatalf("GetSessionInfo accepted the unknown identifier %q", saveSessionID)
			}
		})
	}
	if len(engine.sessions) != 1 {
		t.Errorf("engine holds %d sessions after rejected lookups, want 1", len(engine.sessions))
	}
}

func TestGetSessionInfoReturnsAnIndependentValue(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	mutated, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	mutated.SaveSessionID = "tampered"
	mutated.Platform = "tampered"
	mutated.Format = "tampered"
	mutated.SourcePath = "tampered"
	mutated.SourceKind = "tampered"
	mutated.SaveRevision = "tampered"
	mutated.UnsavedChanges = true

	again, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("second GetSessionInfo: %v", err)
	}
	if again != loaded {
		t.Errorf("changing the returned value changed the stored metadata: %+v, want %+v", again, loaded)
	}
}

// sessionInfoFields is the exact public shape of SessionInfo: the approved
// metadata and nothing else. It is written out by name and type so that adding,
// removing, renaming or retyping a field fails here instead of silently
// widening what a session exposes.
var sessionInfoFields = []struct {
	name string
	kind reflect.Kind
}{
	{"SaveSessionID", reflect.String},
	{"Platform", reflect.String},
	{"Format", reflect.String},
	{"SourcePath", reflect.String},
	{"SourceKind", reflect.String},
	{"SaveRevision", reflect.String},
	{"UnsavedChanges", reflect.Bool},
}

// TestSessionInfoExposesOnlyTheApprovedMetadata pins the public session shape.
// SourcePath and SourceKind are approved source metadata; every other field must
// stay free of the source path, and no field may carry a snapshot byte, a
// handle or an offset.
func TestSessionInfoExposesOnlyTheApprovedMetadata(t *testing.T) {
	sessionInfo := reflect.TypeOf(SessionInfo{})
	if got := sessionInfo.NumField(); got != len(sessionInfoFields) {
		t.Fatalf("SessionInfo has %d fields, want exactly the %d approved ones %v",
			got, len(sessionInfoFields), sessionInfoFields)
	}
	for index, want := range sessionInfoFields {
		field := sessionInfo.Field(index)
		if field.Name != want.name || field.Type.Kind() != want.kind {
			t.Errorf("SessionInfo field %d is %s %s, want %s %s",
				index, field.Name, field.Type.Kind(), want.name, want.kind)
		}
	}
}

// TestGetSessionInfoNeedsNoSourceFileAndExposesNoSnapshot is the protected
// guard on what a public session may reveal, kept at least as strong as it was
// before SourcePath, SourceKind and SaveRevision joined the contract:
//
//   - the source file is removed after LoadSave, so a session that still answers
//     proves the snapshot is private and self-sufficient and that GetSessionInfo
//     reopens nothing;
//   - SourcePath is the approved metadata and must equal the exact load path;
//   - no other field may leak that path, and no field at all may carry a byte of
//     the container, a handle or an offset;
//   - the private snapshot stays in place and untouched behind the session.
func TestGetSessionInfoNeedsNoSourceFileAndExposesNoSnapshot(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	// The session exists, so the source is no longer needed: the metadata comes
	// from the session, not from the file. Removing it also proves that a source
	// which disappears after LoadSave cannot affect the existing session.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove source: %v", err)
	}

	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info != loaded {
		t.Errorf("GetSessionInfo = %+v, want %+v", info, loaded)
	}
	if info.SourcePath != path {
		t.Errorf("sourcePath = %q, want the exact load path %q", info.SourcePath, path)
	}
	if info.SourceKind != string(SourceKindLocal) {
		t.Errorf("sourceKind = %q, want %q", info.SourceKind, SourceKindLocal)
	}
	if info.SaveRevision != "0" {
		t.Errorf("saveRevision = %q, want %q for a freshly loaded session", info.SaveRevision, "0")
	}

	// Only the approved source metadata may name the file. Every other field is
	// checked against the path as well as against the container magic, so no
	// field can smuggle out a snapshot byte, a handle or an offset.
	for name, value := range map[string]string{
		"saveSessionID": info.SaveSessionID,
		"platform":      info.Platform,
		"format":        info.Format,
		"saveRevision":  info.SaveRevision,
	} {
		if strings.Contains(value, path) {
			t.Errorf("%s = %q exposes the source path", name, value)
		}
		if strings.Contains(value, string(pcMagic)) {
			t.Errorf("%s = %q exposes the snapshot content", name, value)
		}
	}
	if strings.Contains(info.SourceKind, path) {
		t.Errorf("sourceKind = %q exposes the source path", info.SourceKind)
	}
	if strings.Contains(info.SourcePath, string(pcMagic)) {
		t.Errorf("sourcePath = %q exposes the snapshot content", info.SourcePath)
	}

	// The snapshot stays private and untouched behind the session.
	stored := engine.sessions[loaded.SaveSessionID]
	if stored == nil || stored.snapshot == nil {
		t.Fatal("GetSessionInfo changed the session or its snapshot")
	}
	// The snapshot is still the whole container the session was created from, so
	// nothing reread, truncated or re-resolved it when the source disappeared.
	head, err := stored.snapshot.readAt(0, magicLength)
	if err != nil || !bytes.Equal(head, pcMagic) {
		t.Fatalf("snapshot head = %x (err %v), want the loaded container magic %x",
			head, err, pcMagic)
	}
}

// TestLoadSaveReportsTheExactSourceOnBothPlatforms covers the source metadata
// and the initial revision for PC and PS4 alike: neither platform gets a
// different contract, and neither path is trimmed, recased or resolved.
func TestLoadSaveReportsTheExactSourceOnBothPlatforms(t *testing.T) {
	for _, testCase := range []struct {
		platform Platform
		format   string
		header   []byte
		size     int64
		// name deliberately carries surrounding spaces and mixed case, so a
		// normalising implementation cannot pass.
		name string
	}{
		{PlatformPC, pcContainerFormat, pcHeader(), pcFixtureSize, " PC Save .sl2 "},
		{PlatformPS4, ps4ContainerFormat, ps4Header(), ps4FixtureSize, " PS4 Save "},
	} {
		t.Run(string(testCase.platform), func(t *testing.T) {
			for _, kind := range []SourceKind{SourceKindLocal, SourceKindTemporary} {
				t.Run(string(kind), func(t *testing.T) {
					path := writeFixture(t, testCase.name, testCase.header, testCase.size)
					engine := New()

					loaded, err := engine.LoadSave(path, "", string(kind))
					if err != nil {
						t.Fatalf("LoadSave: %v", err)
					}
					if loaded.Platform != string(testCase.platform) {
						t.Errorf("platform = %q, want %q", loaded.Platform, testCase.platform)
					}
					if loaded.Format != testCase.format {
						t.Errorf("format = %q, want %q", loaded.Format, testCase.format)
					}
					if loaded.SourcePath != path {
						t.Errorf("sourcePath = %q, want the exact path %q", loaded.SourcePath, path)
					}
					if loaded.SourceKind != string(kind) {
						t.Errorf("sourceKind = %q, want %q", loaded.SourceKind, kind)
					}
					if loaded.SaveRevision != "0" {
						t.Errorf("saveRevision = %q, want %q", loaded.SaveRevision, "0")
					}
					if loaded.UnsavedChanges {
						t.Error("a freshly loaded session reports unsaved changes")
					}
				})
			}
		})
	}
}

// TestLoadSaveRejectsUnknownSourceKind proves the source kind has no default,
// no alias and no tolerance for case or spacing, and that a rejected value
// creates no session and opens nothing.
func TestLoadSaveRejectsUnknownSourceKind(t *testing.T) {
	for _, sourceKind := range []string{
		"", " ", "Local", "LOCAL", " local", "local ", "temp", "Temporary", "remote", "pc",
	} {
		t.Run(sourceKind, func(t *testing.T) {
			path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
			engine := New()

			loaded, err := engine.LoadSave(path, "", sourceKind)
			if err == nil {
				t.Fatalf("LoadSave accepted the unknown source kind %q", sourceKind)
			}
			if loaded != (SessionInfo{}) {
				t.Errorf("LoadSave returned %+v for a rejected source kind, want the zero value", loaded)
			}
			if len(engine.sessions) != 0 {
				t.Errorf("LoadSave left %d session(s) behind for a rejected source kind",
					len(engine.sessions))
			}
		})
	}
}

// TestGetSessionInfoReportsTheCurrentRevisionAfterAMutation covers the revision
// half of the contract at the level the increment is implemented on: a
// committed mutation advances the reported revision and marks the session
// changed, and both values are read back through the public metadata.
func TestGetSessionInfoReportsTheCurrentRevisionAfterAMutation(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	for _, want := range []string{"1", "2", "3"} {
		revision, err := engine.commitRevision(loaded.SaveSessionID, func(*loadedSave) error {
			return nil
		})
		if err != nil {
			t.Fatalf("commitRevision: %v", err)
		}
		if revision != want {
			t.Fatalf("commitRevision = %q, want %q", revision, want)
		}

		info, err := engine.GetSessionInfo(loaded.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.SaveRevision != want {
			t.Errorf("saveRevision = %q, want the revision the mutation returned %q",
				info.SaveRevision, want)
		}
		if !info.UnsavedChanges {
			t.Error("a committed mutation left unsavedChanges false")
		}
		// The source metadata belongs to the session, not to a revision: a
		// mutation must never rewrite or clear it.
		if info.SourcePath != path || info.SourceKind != string(SourceKindLocal) {
			t.Errorf("a mutation changed the source metadata to %q/%q, want %q/%q",
				info.SourcePath, info.SourceKind, path, SourceKindLocal)
		}
	}
}

// TestRejectedMutationChangesNeitherRevisionNorSessionState is the negative
// boundary of the previous test: a refused mutation must leave the revision, the
// change state and the source metadata exactly as they were.
func TestRejectedMutationChangesNeitherRevisionNorSessionState(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// A rejection before any mutation ever succeeded, and one after a mutation
	// did: neither may move the revision.
	for _, want := range []string{"0", "1"} {
		before, err := engine.GetSessionInfo(loaded.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if before.SaveRevision != want {
			t.Fatalf("saveRevision = %q, want %q", before.SaveRevision, want)
		}

		revision, err := engine.commitRevision(loaded.SaveSessionID, func(*loadedSave) error {
			return errors.New("refused by the mutation")
		})
		if err == nil {
			t.Fatal("commitRevision accepted a refused mutation")
		}
		if revision != "" {
			t.Errorf("a refused mutation returned revision %q, want an empty value", revision)
		}

		after, err := engine.GetSessionInfo(loaded.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo after the refusal: %v", err)
		}
		if after != before {
			t.Errorf("a refused mutation changed the session to %+v, want %+v", after, before)
		}

		if _, err := engine.commitRevision(loaded.SaveSessionID, func(*loadedSave) error {
			return nil
		}); err != nil {
			t.Fatalf("commitRevision: %v", err)
		}
	}
}

func TestCloseSessionRemovesTheSessionFromTheEngine(t *testing.T) {
	path := writeFixture(t, "close.sl2", pcHeader(), pcFixtureSize)
	engine := New()
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	if err := engine.CloseSession(loaded.SaveSessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	// The map entry is gone, so the engine holds no reference to the session or
	// to its private snapshot any more.
	if _, exists := engine.sessions[loaded.SaveSessionID]; exists {
		t.Fatal("CloseSession left the session in the engine")
	}
	if _, err := engine.GetSessionInfo(loaded.SaveSessionID); err == nil {
		t.Fatal("GetSessionInfo resolved a closed session")
	}
}

func TestCloseSessionKeepsOtherSessions(t *testing.T) {
	engine := New()
	first, err := engine.LoadSave(writeFixture(t, "first.sl2", pcHeader(), pcFixtureSize), "", "local")
	if err != nil {
		t.Fatalf("LoadSave first: %v", err)
	}
	second, err := engine.LoadSave(writeFixture(t, "second.sl2", pcHeader(), pcFixtureSize), "", "local")
	if err != nil {
		t.Fatalf("LoadSave second: %v", err)
	}

	if err := engine.CloseSession(first.SaveSessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	info, err := engine.GetSessionInfo(second.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo of the untouched session: %v", err)
	}
	if info != second {
		t.Errorf("GetSessionInfo = %+v, want %+v", info, second)
	}
}

func TestCloseSessionRejectsEmptyIdentifier(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeFixture(t, "empty-id.sl2", pcHeader(), pcFixtureSize), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	if err := engine.CloseSession(""); err == nil {
		t.Fatal("CloseSession accepted an empty identifier")
	}
	if _, exists := engine.sessions[loaded.SaveSessionID]; !exists {
		t.Fatal("a rejected CloseSession removed a session")
	}
}

func TestCloseSessionRejectsUnknownIdentifier(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeFixture(t, "unknown-id.sl2", pcHeader(), pcFixtureSize), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// A foreign identifier and a near miss of the real one are both unknown: the
	// identifier is matched exactly, without trimming, normalising or guessing.
	for _, saveSessionID := range []string{
		"00000000000000000000000000000000",
		loaded.SaveSessionID + " ",
		" " + loaded.SaveSessionID,
	} {
		t.Run(saveSessionID, func(t *testing.T) {
			if err := engine.CloseSession(saveSessionID); err == nil {
				t.Fatalf("CloseSession accepted the unknown identifier %q", saveSessionID)
			}
			if _, exists := engine.sessions[loaded.SaveSessionID]; !exists {
				t.Fatal("a rejected CloseSession removed the existing session")
			}
		})
	}

	// Closing twice is rejected the second time: a closed session is never
	// resolved again.
	if err := engine.CloseSession(loaded.SaveSessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	if err := engine.CloseSession(loaded.SaveSessionID); err == nil {
		t.Fatal("CloseSession accepted an already closed session")
	}
}
