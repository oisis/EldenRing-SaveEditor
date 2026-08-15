package schema

// TutorialDocument declares one user-facing tutorial whose title is present in
// the official English TutorialTitle FMG. TutorialID is the TutorialParam row ID
// stored in the save's TutorialData list after the tutorial has been unlocked.
// Both facts retain their independent source provenance.
type TutorialDocument struct {
	TutorialID Fact[uint32] `json:"tutorialID"`
	Title      Fact[string] `json:"title"`
}
