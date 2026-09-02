package saveengine

import "strconv"

// This file owns the one backend event of stage 3c: session.changed, the
// notification that a save session committed a mutation.
//
// The engine publishes it, it never emits it: SaveEngine must not know Wails,
// so the host registers a sink and the desktop layer turns the event into a
// Wails emission. The engine guarantees three things about that sink:
//
//   - it is called exactly once per committed MutationReceipt, and never for a
//     rejected mutation, a rollback or a success that committed nothing;
//   - it is called in commit order, so the sequence numbers a subscriber sees
//     are the order the revisions were created in;
//   - it is never called while Engine.mutex is held, so a sink may take its own
//     time, block, or call back into the host without deadlocking the engine.

// SessionChangedEventName is the stable name of the event. It is the value the
// desktop host emits under and the frontend subscribes to, so both sides name
// it once from here.
const SessionChangedEventName = "session.changed"

// SessionChangedEvent notifies subscribers that one committed mutation advanced
// the revision of a save session. It carries exactly the receipt of that
// mutation plus the sequence number of this session's event stream.
//
// The event is a notification and not a state document: a subscriber refreshes
// the getters named by ChangedScopes and never reconstructs save data from it.
//
// Sequence is the canonical decimal rendering of the session's private event
// counter, a string for the same reason SaveRevision is one: no consumer may
// round it to a JavaScript number, increment it or reorder it. It starts at "1"
// for the first committed mutation of a session and is monotonic within that
// session; two sessions number their events independently.
type SessionChangedEvent struct {
	Sequence      string   `json:"sequence"`
	OperationID   string   `json:"operationID"`
	OperationKind string   `json:"operationKind"`
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	ChangedScopes []string `json:"changedScopes"`
}

// SessionChangedSink receives one published event. It runs on the goroutine of
// the mutation that produced the event, outside Engine.mutex.
type SessionChangedSink func(SessionChangedEvent)

// SetSessionChangedSink installs the single subscriber of committed session
// mutations. A nil sink disables publication; installing a sink replaces the
// previous one and never replays events published before it was installed.
//
// ponytail: one sink, not a subscriber registry. The desktop host is the only
// consumer and it fans out to the frontend itself; a registry would be a second
// dispatch layer for a single caller.
func (engine *Engine) SetSessionChangedSink(sink SessionChangedSink) {
	engine.eventMutex.Lock()
	defer engine.eventMutex.Unlock()
	engine.sessionChangedSink = sink
}

// enqueueSessionChanged records one committed event. The caller must hold
// Engine.mutex, which is what makes the queue order the commit order.
func (engine *Engine) enqueueSessionChanged(event SessionChangedEvent) {
	engine.eventMutex.Lock()
	defer engine.eventMutex.Unlock()
	engine.eventQueue = append(engine.eventQueue, event)
}

// enqueueCommitted queues the session.changed event of one committed receipt.
// It is the single place the event is built, so the event of every commit path
// carries exactly the receipt's identifiers and exactly its scopes. The caller
// must hold Engine.mutex.
func (engine *Engine) enqueueCommitted(session *Session, receipt MutationReceipt) {
	engine.enqueueSessionChanged(SessionChangedEvent{
		Sequence:      session.nextEventSequence(),
		OperationID:   receipt.OperationID,
		OperationKind: receipt.OperationKind,
		SaveSessionID: receipt.SaveSessionID,
		SaveRevision:  receipt.SaveRevision,
		// The scopes are copied so the event and the receipt can never alias one
		// slice across the lock boundary; the values are identical.
		ChangedScopes: append([]string(nil), receipt.ChangedScopes...),
	})
}

// publishSessionChanged delivers every queued event in order. The caller must
// no longer hold Engine.mutex.
//
// ponytail: delivery runs on the caller's goroutine and starts no goroutine of
// its own, so there is nothing to leak and nothing to shut down. A slow sink
// therefore slows the mutation that happens to be draining; that is acceptable
// while the only sink is one Wails emission, and a queue plus one worker is the
// upgrade path if a sink ever becomes expensive.
func (engine *Engine) publishSessionChanged() {
	engine.eventDrainMutex.Lock()
	defer engine.eventDrainMutex.Unlock()
	for {
		engine.eventMutex.Lock()
		if len(engine.eventQueue) == 0 {
			engine.eventMutex.Unlock()
			return
		}
		event := engine.eventQueue[0]
		engine.eventQueue = engine.eventQueue[1:]
		sink := engine.sessionChangedSink
		engine.eventMutex.Unlock()

		if sink != nil {
			sink(event)
		}
	}
}

// nextEventSequence advances and renders this session's event counter. The
// caller must already hold Engine.mutex, like every other session helper.
func (session *Session) nextEventSequence() string {
	session.eventSeq++
	return session.eventSequenceString()
}

// eventSequenceString is the public rendering of the private event counter: a
// canonical decimal string, "0" for a session that has published no event yet.
func (session *Session) eventSequenceString() string {
	return strconv.FormatUint(session.eventSeq, 10)
}
