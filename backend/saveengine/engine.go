// Package saveengine is the sole owner of save reading in SaveForge 2.0. The
// current stage is read-only: it opens a local save, recognises its container,
// validates the structure the stage depends on and creates a session. It never
// modifies the file it opened.
package saveengine

import (
	"fmt"
	"sync"
)

// magicLength is the number of leading bytes needed to recognise a container.
// Both supported magics are four bytes long.
const magicLength = 4

// Engine owns the read-only save sessions of one backend instance. Sessions are
// kept under their own identifier because a later GetLoadedSave reads a session
// by its saveSessionID.
type Engine struct {
	mutex    sync.Mutex
	sessions map[string]*loadedSave
}

// loadedSave is the private state of one session: its metadata model and the
// private, read-only snapshot of the file it was created from. Nothing here
// leaves the package; a later GetLoadedSave will read the snapshot through the
// engine instead of exposing it.
type loadedSave struct {
	session  *Session
	snapshot *codec
}

// New returns an engine holding no session.
func New() *Engine {
	return &Engine{sessions: make(map[string]*loadedSave)}
}

// LoadSave reads the local file at path read-only, recognises its platform,
// validates it against expectedPlatform, validates the container structure and
// registers a new session together with a private snapshot of the file. The
// file is never written to, it is closed before this function returns, and no
// session is created when any step fails.
//
// expectedPlatform accepts "", "pc" and "ps4". The value is matched exactly and
// case-sensitively and is never trimmed; an empty value expresses no
// expectation. Any other value is an error, and a recognised platform differing
// from a non-empty expectation is rejected before a session exists.
func (engine *Engine) LoadSave(path string, expectedPlatform string) (SessionInfo, error) {
	expected, err := parseExpectedPlatform(expectedPlatform)
	if err != nil {
		return SessionInfo{}, err
	}

	// The codec snapshots the file and closes it before returning, so the engine
	// never holds a handle to the user's file and the session data cannot change
	// underneath it.
	source, err := openCodec(path)
	if err != nil {
		return SessionInfo{}, err
	}

	head, err := source.readAt(0, magicLength)
	if err != nil {
		return SessionInfo{}, errUnsupportedContainer
	}

	var platform Platform
	var format string
	switch {
	case pcRecognises(head):
		platform, format = PlatformPC, pcContainerFormat
	case ps4Recognises(head):
		platform, format = PlatformPS4, ps4ContainerFormat
	default:
		return SessionInfo{}, errUnsupportedContainer
	}

	if expected != "" && expected != platform {
		return SessionInfo{}, fmt.Errorf("expected a %s save, but the file is a %s save", expected, platform)
	}

	switch platform {
	case PlatformPC:
		err = pcValidate(source)
	case PlatformPS4:
		err = ps4Validate(source)
	}
	if err != nil {
		return SessionInfo{}, err
	}

	session, err := newSession(platform, format)
	if err != nil {
		return SessionInfo{}, err
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	engine.sessions[session.id] = &loadedSave{session: session, snapshot: source}

	return session.Info(), nil
}

// errUnsupportedContainer rejects every input that is not an unambiguous native
// PC or PS4 container. An encrypted or unknown container is never decrypted and
// never guessed.
var errUnsupportedContainer = fmt.Errorf(
	"unsupported save container: the file is neither a native PC nor a native PS4 save")

// parseExpectedPlatform validates the caller's platform expectation.
func parseExpectedPlatform(value string) (Platform, error) {
	switch value {
	case "":
		return "", nil
	case string(PlatformPC):
		return PlatformPC, nil
	case string(PlatformPS4):
		return PlatformPS4, nil
	default:
		return "", fmt.Errorf("unknown expected platform %q, want %q, %q or an empty value",
			value, PlatformPC, PlatformPS4)
	}
}
