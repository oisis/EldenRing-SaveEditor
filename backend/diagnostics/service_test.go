package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fixedClock keeps the timestamps deterministic without making the tests wait.
func fixedClock() func() time.Time {
	moment := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	return func() time.Time {
		moment = moment.Add(time.Second)
		return moment
	}
}

func severities(records []Record) []string {
	found := make([]string, 0, len(records))
	for _, record := range records {
		found = append(found, record.Severity)
	}
	return found
}

// Debug Mode gates debug records only. Turning it off must stop new debug
// records without removing what was already recorded and without silencing the
// three base severities.
func TestDebugModeGatesOnlyDebugRecordsAndKeepsWhatWasRecorded(t *testing.T) {
	t.Parallel()

	service := NewService(Options{Now: fixedClock()})

	service.Log(Entry{Event: EventOperationStarted, Operation: OperationSave})
	service.Log(Entry{Event: EventOperationFinished, Operation: OperationSave, Status: StatusSucceeded})

	page, err := service.Records("", 50, "")
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	if got := severities(page.Records); len(got) != 1 || got[0] != SeverityInfo {
		t.Fatalf("with Debug Mode off, severities = %v, want exactly one info record", got)
	}

	if state := service.SetDebugMode(true); !state.Enabled {
		t.Fatalf("SetDebugMode(true) reported enabled = false")
	}
	service.Log(Entry{Event: EventOperationStarted, Operation: OperationSave})
	service.Log(Entry{Event: EventOperationFinished, Operation: OperationSave, Status: StatusFailed})

	page, err = service.Records("", 50, "")
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	want := []string{SeverityInfo, SeverityInfo, SeverityDebug, SeverityError}
	got := severities(page.Records)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("severities = %v, want %v", got, want)
	}

	if state := service.SetDebugMode(false); state.Enabled {
		t.Fatalf("SetDebugMode(false) reported enabled = true")
	}
	service.Log(Entry{Event: EventOperationStarted, Operation: OperationSave})

	page, err = service.Records("", 50, "")
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	// The four earlier records survive, the mode change is recorded as info and
	// the new debug record is not written.
	want = []string{SeverityInfo, SeverityInfo, SeverityDebug, SeverityError, SeverityInfo}
	if got := severities(page.Records); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("after disabling, severities = %v, want %v", got, want)
	}
}

// An entry naming anything outside the closed catalogue records nothing at all.
// This is the boundary that keeps a path, a host or a raw error out of the
// buffer, the file and the report, so it fails closed rather than sanitising.
func TestEntriesOutsideTheClosedCatalogueAreRejectedEntirely(t *testing.T) {
	t.Parallel()

	service := NewService(Options{Now: fixedClock()})
	service.SetDebugMode(true)

	rejected := []Entry{
		{Event: "user_opened_/Users/oisis/save.sl2"},
		{Event: EventOperationFinished, Status: "ssh://deck@192.168.0.10"},
		{Event: EventOperationStarted, Operation: "connect steam_76561198000000000"},
		{Event: EventOperationStageFinished, Stage: "/private/tmp/staging"},
		{Event: EventOperationFinished, Status: StatusFailed, Code: "dial tcp: no route to host"},
		{Event: EventOperationFinished, Status: StatusSucceeded, TargetState: "/home/deck"},
		{Event: EventOperationStarted, CorrelationID: "76561198000000000"},
		{Event: EventApplicationStarted, Version: "/Users/private/save"},
		{Event: EventApplicationStarted, Build: "ssh://secret@host"},
		{Event: EventApplicationStarted, Platform: "private-host"},
	}
	for _, entry := range rejected {
		service.Log(entry)
	}

	page, err := service.Records("", 50, "")
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	// One record exists: the info record SetDebugMode produced. Nothing else was
	// accepted.
	if len(page.Records) != 1 || page.Records[0].Event != EventDiagnosticModeChanged {
		t.Fatalf("records = %+v, want only the diagnostic_mode_changed record", page.Records)
	}
	if state := service.State(); state.DroppedRecords != len(rejected) {
		t.Fatalf("DroppedRecords = %d, want %d", state.DroppedRecords, len(rejected))
	}
}

// The cursor is the sequence of the last record already seen, so an incremental
// reader never sees the same record twice and an evicted cursor is reported
// rather than answered with a silent gap.
func TestRecordsAreReadIncrementallyAndAnExpiredCursorIsReported(t *testing.T) {
	t.Parallel()

	service := NewService(Options{Now: fixedClock()})

	service.Log(Entry{Event: EventApplicationStarted, Version: "2.0.0", Platform: "darwin/arm64"})
	first, err := service.Records("", 50, "")
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	if len(first.Records) != 1 {
		t.Fatalf("first page = %d records, want 1", len(first.Records))
	}

	repeated, err := service.Records(first.NextCursor, 50, "")
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	if len(repeated.Records) != 0 {
		t.Fatalf("continuing from the cursor returned %d records, want 0", len(repeated.Records))
	}

	service.Log(Entry{Event: EventOperationFinished, Operation: OperationSave, Status: StatusSucceeded})
	next, err := service.Records(first.NextCursor, 50, "")
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	if len(next.Records) != 1 || next.Records[0].Event != EventOperationFinished {
		t.Fatalf("next page = %+v, want only the new record", next.Records)
	}

	// The buffer is bounded, so a reader that fell far enough behind is told its
	// cursor expired instead of being handed an incomplete continuation.
	for index := 0; index < bufferCapacity+10; index++ {
		service.Log(Entry{
			Event: EventOperationFinished, Operation: OperationSave, Status: StatusSucceeded,
		})
	}
	stale, err := service.Records(first.NextCursor, 50, "")
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	if !stale.CursorExpired {
		t.Fatalf("CursorExpired = false, want true for an evicted cursor")
	}
	if stale.TotalBuffered != bufferCapacity {
		t.Fatalf("TotalBuffered = %d, want the buffer ceiling %d", stale.TotalBuffered, bufferCapacity)
	}
	recent := service.RecentRecords(200)
	if len(recent) != 200 || recent[0].Seq != 313 || recent[199].Seq != 512 {
		t.Fatalf("report slice must contain the newest 200 records: %+v", recent)
	}
	maxCursor, err := service.Records("18446744073709551615", 50, "")
	if err != nil || maxCursor.CursorExpired || len(maxCursor.Records) != 0 {
		t.Fatalf("maximum cursor must not overflow into an expired cursor: %+v, %v", maxCursor, err)
	}
}

// Rotation is bounded and touches only the files this package created. A file
// that merely lives in the same directory must survive untouched.
func TestRotationIsBoundedAndLeavesForeignFilesAlone(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	foreign := filepath.Join(directory, "notes.txt")
	if err := os.WriteFile(foreign, []byte("keep me"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	service := &Service{now: time.Now, directory: directory}
	// Every generation is created full, so the next rotation must drop exactly
	// one of them rather than growing the set.
	for generation := 0; generation < logFileCount; generation++ {
		name := filepath.Join(directory, rotatedName(generation))
		if err := os.WriteFile(name, []byte(strings.Repeat("x", 16)), 0o600); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}
	if err := service.rotate(); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	logs := 0
	foreignFound := false
	for _, entry := range entries {
		switch {
		case entry.Name() == "notes.txt":
			foreignFound = true
		case strings.HasPrefix(entry.Name(), "saveforge-diagnostics"):
			logs++
		default:
			t.Errorf("rotation produced an unexpected file %q", entry.Name())
		}
	}
	if !foreignFound {
		t.Errorf("rotation removed the unrelated file in the log directory")
	}
	// The active name is now free for the next open, so one generation less than
	// the ceiling remains on disk.
	if logs != logFileCount-1 {
		t.Errorf("log files after rotation = %d, want %d", logs, logFileCount-1)
	}
	if _, err := os.Stat(filepath.Join(directory, rotatedName(0))); !os.IsNotExist(err) {
		t.Errorf("the active file was not rotated away")
	}
}

// A sink that cannot write must not change what an operation did. The records
// keep reaching the memory buffer and the failure is reported as state, never
// raised at the caller and never logged as another record.
func TestASinkFailureIsReportedWithoutAffectingRecordsOrCallers(t *testing.T) {
	t.Parallel()

	// A regular file in place of the log directory makes every open fail.
	blocked := filepath.Join(t.TempDir(), "logs")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	service := NewService(Options{Directory: blocked, Now: fixedClock()})
	service.Log(Entry{Event: EventApplicationStarted, Version: "2.0.0"})
	service.Log(Entry{Event: EventOperationFinished, Operation: OperationSave, Status: StatusSucceeded})
	service.Close()

	page, err := service.Records("", 50, "")
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	if len(page.Records) != 2 {
		t.Fatalf("buffered records = %d, want 2 despite the failing sink", len(page.Records))
	}
	state := service.State()
	if state.LocalLoggingAvailable {
		t.Errorf("LocalLoggingAvailable = true, want false after the sink failed")
	}
	if state.LogDirectoryExists {
		t.Errorf("LogDirectoryExists = true for a regular file")
	}
	if state.DroppedRecords == 0 {
		t.Errorf("DroppedRecords = 0, want the lost records to be counted")
	}
	// No record describes the logger's own failure: that is what would loop.
	for _, record := range page.Records {
		if record.Event != EventApplicationStarted && record.Event != EventOperationFinished {
			t.Errorf("the sink failure produced an extra record %q", record.Event)
		}
	}
}

// A working sink writes one JSON object per line and nothing else.
func TestTheLocalSinkWritesOneSafeJSONLinePerRecord(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "logs")
	service := NewService(Options{Directory: directory, Now: fixedClock()})
	service.Log(Entry{Event: EventApplicationStarted, Version: "2.0.0", Platform: "darwin/arm64"})
	service.Log(Entry{
		Event: EventOperationFinished, Operation: OperationDeployToTarget,
		Status: StatusSucceeded, CorrelationID: "0123456789abcdef",
	})
	service.Log(Entry{Event: EventOperationStarted, CorrelationID: "/private/secret"})
	service.Close()
	service.Log(Entry{Event: EventApplicationStarted, Version: "2.0.0"})

	written, err := os.ReadFile(filepath.Join(directory, logFileName))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(string(written), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("wrote %d lines, want 2", len(lines))
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
			t.Errorf("line %q is not one JSON object", line)
		}
	}
	if info, err := os.Stat(filepath.Join(directory, logFileName)); err != nil {
		t.Fatalf("Stat: %v", err)
	} else if info.Mode().Perm() != 0o600 {
		t.Errorf("log file mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestCloseAndRotationKeepTheLogBounded(t *testing.T) {
	directory := t.TempDir()
	active := filepath.Join(directory, logFileName)
	if err := os.WriteFile(active, []byte("previous log"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(active, logFileMaxBytes-1); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{Directory: directory})
	service.Log(Entry{Event: EventApplicationStarted, Version: "2.0.0"})
	service.Close()
	if info, err := os.Stat(active); err != nil || info.Size() >= logFileMaxBytes {
		t.Fatalf("active file exceeded the limit after reopening: %v, %v", info, err)
	}
	if _, err := os.Stat(filepath.Join(directory, rotatedName(1))); err != nil {
		t.Fatal(err)
	}
	// Logging during or after shutdown must never send to a closed channel.
	service = NewService(Options{Directory: directory})
	var writers sync.WaitGroup
	for i := 0; i < 4; i++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for j := 0; j < 10; j++ {
				service.Log(Entry{Event: EventApplicationStarted})
			}
		}()
	}
	service.Close()
	writers.Wait()
}
