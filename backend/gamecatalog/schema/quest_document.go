package schema

// QuestFlag is a single target event flag state for a quest step.
type QuestFlag struct {
	ID    uint32 `json:"id"`
	Value bool   `json:"value"`
}

// QuestStepDocument represents one supported step in an NPC questline.
type QuestStepDocument struct {
	Key         string       `json:"key"`
	Description Fact[string] `json:"description"`
	Location    Fact[string] `json:"location"`
	Flags       []QuestFlag  `json:"flags"`
}

// QuestDocument declares one NPC questline and its supported steps.
type QuestDocument struct {
	Name  Fact[string]        `json:"name"`
	Steps []QuestStepDocument `json:"steps"`
}
