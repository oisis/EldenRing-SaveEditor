package saveengine

import (
	"strings"
	"sync"
	"testing"
)

// These tests own the publication contract of session.changed: one event per
// committed receipt, none for anything else, in commit order, and never under
// the engine lock.

func loadSessionEventFixture(t *testing.T) (*Engine, string) {
	t.Helper()

	engine := New()
	loaded, err := engine.LoadSave(writeCharacterNameFixture(
		t, PlatformPC, true, true, "Ranni", "Ranni"), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

// recordSessionEvents installs a sink collecting every published event.
func recordSessionEvents(engine *Engine) (*[]SessionChangedEvent, *sync.Mutex) {
	var mutex sync.Mutex
	events := make([]SessionChangedEvent, 0, 8)
	engine.SetSessionChangedSink(func(event SessionChangedEvent) {
		mutex.Lock()
		defer mutex.Unlock()
		events = append(events, event)
	})
	return &events, &mutex
}

func TestCommittedMutationPublishesExactlyOneSessionChangedEvent(t *testing.T) {
	engine, saveSessionID := loadSessionEventFixture(t)
	events, mutex := recordSessionEvents(engine)

	result, err := engine.SetCharacterName(saveSessionID, setActiveTestSlot, "Melina", "0")
	if err != nil {
		t.Fatalf("SetCharacterName: %v", err)
	}

	mutex.Lock()
	defer mutex.Unlock()
	if len(*events) != 1 {
		t.Fatalf("published %d events, want exactly one", len(*events))
	}
	event := (*events)[0]
	if event.Sequence != "1" {
		t.Errorf("sequence = %q, want the first event of the session", event.Sequence)
	}
	if event.OperationID != result.OperationID || event.OperationKind != result.OperationKind {
		t.Errorf("event = %+v, want the identifiers of receipt %+v", event, result.MutationReceipt)
	}
	if event.SaveSessionID != saveSessionID || event.SaveRevision != result.SaveRevision {
		t.Errorf("event = %+v, want session %q at revision %q",
			event, saveSessionID, result.SaveRevision)
	}
	if strings.Join(event.ChangedScopes, ",") != strings.Join(result.ChangedScopes, ",") {
		t.Errorf("changedScopes = %v, want exactly the receipt's %v",
			event.ChangedScopes, result.ChangedScopes)
	}
	// The event must not alias the receipt's slice: two owners of one slice
	// across the lock boundary would let either of them rewrite the other.
	if len(event.ChangedScopes) > 0 {
		event.ChangedScopes[0] = "mutated"
		if result.ChangedScopes[0] == "mutated" {
			t.Error("the event and the receipt share one changedScopes slice")
		}
	}
}

// The sequence counts committed mutations of one session and is reported by
// SessionInfo, so a subscriber can resynchronise against it.
func TestSessionEventSequenceIsMonotonicAndReadableFromTheSession(t *testing.T) {
	engine, saveSessionID := loadSessionEventFixture(t)
	events, mutex := recordSessionEvents(engine)

	info, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.EventSequence != "0" {
		t.Errorf("eventSequence = %q, want %q for a freshly loaded session",
			info.EventSequence, "0")
	}

	for index, revision := range []string{"0", "1", "2"} {
		if _, err := engine.SetCharacterRunes(
			saveSessionID, setActiveTestSlot, uint32(index+1), revision); err != nil {
			t.Fatalf("SetCharacterRunes at revision %q: %v", revision, err)
		}
	}

	mutex.Lock()
	sequences := make([]string, 0, len(*events))
	for _, event := range *events {
		sequences = append(sequences, event.Sequence)
	}
	mutex.Unlock()

	if strings.Join(sequences, ",") != "1,2,3" {
		t.Errorf("sequences = %v, want 1,2,3 in commit order", sequences)
	}
	info, err = engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after three commits: %v", err)
	}
	if info.EventSequence != "3" {
		t.Errorf("eventSequence = %q, want %q", info.EventSequence, "3")
	}
}

// A rejection, a rollback and a success that commits nothing all leave the
// stream untouched: no event, and no sequence movement a later subscriber could
// mistake for a lost event.
func TestRejectedRolledBackAndNoCommitOutcomesPublishNothing(t *testing.T) {
	engine, saveSessionID := loadSessionEventFixture(t)
	events, mutex := recordSessionEvents(engine)

	if _, err := engine.SetCharacterName(
		saveSessionID, setActiveTestSlot, "Melina", "7"); err == nil {
		t.Fatal("a stale revision was accepted")
	}
	if _, err := engine.SetCharacterName(
		saveSessionID, setActiveTestSlot, strings.Repeat("x", 64), "0"); err == nil {
		t.Fatal("an invalid name was accepted")
	}
	// The slot is already active, so this is a domain success that commits
	// nothing.
	unchanged, err := engine.SetCharacterActive(saveSessionID, setActiveTestSlot, true, "0")
	if err != nil {
		t.Fatalf("idempotent SetCharacterActive: %v", err)
	}
	if unchanged.Changed {
		t.Fatalf("result = %+v, want an unchanged activity request", unchanged)
	}

	mutex.Lock()
	defer mutex.Unlock()
	if len(*events) != 0 {
		t.Errorf("published %v, want no event at all", *events)
	}
	info, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.EventSequence != "0" || info.SaveRevision != "0" || info.UnsavedChanges {
		t.Errorf("session = %+v, want an untouched session", info)
	}
}

// The sink is the host's callback, so it must never run under the engine lock.
// A sink that calls a public engine method proves it: under the lock this would
// deadlock instead of returning.
func TestSessionChangedSinkRunsOutsideTheEngineLock(t *testing.T) {
	engine, saveSessionID := loadSessionEventFixture(t)

	var reentrant SessionInfo
	var reentrantErr error
	engine.SetSessionChangedSink(func(event SessionChangedEvent) {
		reentrant, reentrantErr = engine.GetSessionInfo(event.SaveSessionID)
	})

	if _, err := engine.SetCharacterName(saveSessionID, setActiveTestSlot, "Melina", "0"); err != nil {
		t.Fatalf("SetCharacterName: %v", err)
	}
	if reentrantErr != nil {
		t.Fatalf("the sink could not read the session: %v", reentrantErr)
	}
	if reentrant.SaveRevision != "1" || reentrant.EventSequence != "1" {
		t.Errorf("session seen by the sink = %+v, want the committed revision 1", reentrant)
	}
}

// Concurrent commits must reach the sink in commit order. The sequence numbers
// are assigned under the engine lock, so an out-of-order delivery would show up
// here as a sequence that goes backwards.
func TestConcurrentCommitsPublishInCommitOrder(t *testing.T) {
	engine, saveSessionID := loadSessionEventFixture(t)
	events, mutex := recordSessionEvents(engine)

	// Every writer retries until its revision matches, so the commits interleave
	// but each one still advances the revision by exactly one.
	const writers = 8
	var wait sync.WaitGroup
	wait.Add(writers)
	for index := 0; index < writers; index++ {
		go func(runes uint32) {
			defer wait.Done()
			for {
				info, err := engine.GetSessionInfo(saveSessionID)
				if err != nil {
					return
				}
				if _, err := engine.SetCharacterRunes(
					saveSessionID, setActiveTestSlot, runes, info.SaveRevision); err == nil {
					return
				}
			}
		}(uint32(index + 1))
	}
	wait.Wait()

	mutex.Lock()
	defer mutex.Unlock()
	if len(*events) != writers {
		t.Fatalf("published %d events, want one per commit (%d)", len(*events), writers)
	}
	previous := ""
	for _, event := range *events {
		if previous != "" && !(len(previous) < len(event.Sequence) ||
			(len(previous) == len(event.Sequence) && previous < event.Sequence)) {
			t.Fatalf("sequence %q was delivered after %q", event.Sequence, previous)
		}
		previous = event.Sequence
	}
}
