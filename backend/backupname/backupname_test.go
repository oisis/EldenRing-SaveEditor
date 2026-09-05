package backupname

import (
	"testing"
	"time"
)

var when = time.Date(2026, 9, 5, 10, 30, 0, 0, time.UTC)

// TestCandidateRendersTheDefaultAndCustomPatterns covers the whole grammar in
// one flow: the default name 2.0 has always written, a custom pattern, and the
// collision counter that stays in front of the suffix.
func TestCandidateRendersTheDefaultAndCustomPatterns(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		pattern   string
		collision int
		want      string
	}{
		{"the default pattern", Default, 1, "ER0000.sl2.20260905103000_bak"},
		{"the default pattern on a collision", Default, 3, "ER0000.sl2.20260905103000_3_bak"},
		{"a custom order", "{timestamp}-{filename}", 1, "20260905103000-ER0000.sl2_bak"},
		{"literal text", "backup of {filename} at {timestamp}", 1,
			"backup of ER0000.sl2 at 20260905103000_bak"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := Candidate(testCase.pattern, "/saves/ER0000.sl2", when, testCase.collision)
			if err != nil {
				t.Fatalf("Candidate: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("Candidate = %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestValidateRefusesEveryUnsafePattern: the backend is the source of the rules,
// and each of these would let a backup name something other than a file beside
// the save.
func TestValidateRefusesEveryUnsafePattern(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		pattern string
	}{
		{"empty", ""},
		{"no file name token", "{timestamp}"},
		{"no timestamp token", "{filename}"},
		{"the file name twice", "{filename}-{filename}-{timestamp}"},
		{"an unknown token", "{filename}-{timestamp}-{user}"},
		{"a POSIX separator", "backups/{filename}.{timestamp}"},
		{"a Windows separator", `backups\{filename}.{timestamp}`},
		{"a path traversal", "../{filename}.{timestamp}"},
		{"a control character", "{filename}\n{timestamp}"},
		{"a colon Windows cannot carry", "{filename}:{timestamp}"},
		{"a leading dot", ".{filename}.{timestamp}"},
		{"a reserved Windows device", "NUL.{filename}{timestamp}"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := Validate(testCase.pattern); err == nil {
				t.Fatalf("Validate(%q) accepted an unsafe pattern", testCase.pattern)
			}
			if Example(testCase.pattern) != "" {
				t.Fatalf("Example(%q) offered a preview of a refused pattern", testCase.pattern)
			}
		})
	}
	if err := Validate(Default); err != nil {
		t.Fatalf("Validate(Default) = %v", err)
	}
	if got := Example(Default); got != "ER0000.sl2.20260824202530_bak" {
		t.Fatalf("Example(Default) = %q, want the documented default name", got)
	}
	if Normalise("  ") != Default {
		t.Fatal("a configuration without a pattern does not fall back to the default")
	}
}

// TestMatchesDefaultRecognisesOnlyTheFixedGrammar: retention uses this to see
// backups written before the pattern became configurable, so it must not claim
// somebody else's file.
func TestMatchesDefaultRecognisesOnlyTheFixedGrammar(t *testing.T) {
	if _, ok := MatchesDefault("ER0000.sl2.20260905103000_bak", "ER0000.sl2"); !ok {
		t.Fatal("the default name was not recognised")
	}
	if _, ok := MatchesDefault("ER0000.sl2.20260905103000_2_bak", "ER0000.sl2"); !ok {
		t.Fatal("the default name with a collision counter was not recognised")
	}
	for _, name := range []string{
		"ER0000.sl2.manual.bak",
		"ER0000.sl2.20260905103000_bak.old",
		"other.sl2.20260905103000_bak",
		"saveforge-20260905103000-ER0000.sl2_bak",
	} {
		if _, ok := MatchesDefault(name, "ER0000.sl2"); ok {
			t.Fatalf("%q was claimed as an automatic backup of ER0000.sl2", name)
		}
	}
}
