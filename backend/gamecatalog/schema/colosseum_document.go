package schema

// ColosseumDocument declares one colosseum arena. Name is the arena name the
// game itself shows and UnlockEventFlagID is the event flag whose set bit
// records the unlock in a save. The two facts come from different sources and
// therefore carry separate provenance; neither is derived from the other.
type ColosseumDocument struct {
	Name              Fact[string] `json:"name"`
	UnlockEventFlagID Fact[uint32] `json:"unlockEventFlagID"`
}
