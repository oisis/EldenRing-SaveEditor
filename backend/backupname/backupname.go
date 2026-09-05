// Package backupname owns the grammar of automatic backup file names.
//
// It is the single source of truth for that grammar. The local Save lifecycle
// and the deployment target backups both render their names here, so a user
// reads one convention everywhere and one validator decides what a pattern may
// contain.
package backupname

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The grammar. It is deliberately closed: exactly two tokens exist and both are
// mandatory, so every rendered name still states which file it protects and
// when it was taken.
const (
	// FilenameToken expands to the source file name with its extension and
	// without any directory part.
	FilenameToken = "{filename}"
	// TimestampToken expands to the creation time in TimestampLayout.
	TimestampToken = "{timestamp}"
	// TimestampLayout is the format 2.0 has always written and the one the
	// existing backups on disk carry.
	TimestampLayout = "20060102150405"
	// Default is the pattern a host that configured none runs under.
	Default = "{filename}.{timestamp}"
	// Suffix is appended by the backend and is not part of the pattern. It is
	// what makes a backup recognisable as one.
	Suffix = "_bak"
	// maximumNameLength keeps a rendered name inside the shortest file name
	// limit of the systems this application supports.
	maximumNameLength = 200
)

// sampleFileName and sampleTime produce the example the Settings screen shows.
// They are fixed so the preview of one pattern never changes between reads.
const sampleFileName = "ER0000.sl2"

var sampleTime = time.Date(2026, 8, 24, 20, 25, 30, 0, time.UTC)

// forbidden are the characters no supported system carries in a file name, plus
// the two path separators. A pattern that contains one cannot address a file
// beside the save, so it is refused rather than sanitised.
const forbidden = `/\:*?"<>|`

// windowsDeviceNames are reserved on Windows whatever extension follows them.
var windowsDeviceNames = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true,
	"com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true,
	"lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// Validate reports whether a pattern may be stored.
//
// It fails closed: an unknown token, a separator, a control character or a name
// that is not portable between the supported systems is refused, and so is a
// pattern whose expansion would be one of those.
func Validate(pattern string) error {
	if pattern == "" {
		return errors.New("the backup name pattern is empty")
	}
	if strings.Count(pattern, FilenameToken) != 1 {
		return fmt.Errorf("the backup name pattern must contain %s exactly once", FilenameToken)
	}
	if strings.Count(pattern, TimestampToken) != 1 {
		return fmt.Errorf("the backup name pattern must contain %s exactly once", TimestampToken)
	}
	// Every remaining brace belongs to a token this grammar does not define.
	literal := strings.NewReplacer(FilenameToken, "", TimestampToken, "").Replace(pattern)
	if strings.ContainsAny(literal, "{}") {
		return fmt.Errorf(
			"the backup name pattern accepts only %s and %s", FilenameToken, TimestampToken)
	}
	if err := checkCharacters("pattern", literal); err != nil {
		return err
	}
	// The literal text is safe on its own; the expansion still has to be.
	_, err := Render(pattern, sampleFileName, sampleTime)
	return err
}

// Render expands one pattern into the base name of a backup. The suffix and the
// collision counter are added by Candidate, not here.
func Render(pattern string, fileName string, when time.Time) (string, error) {
	base := path.Base(fileName)
	if base == "." || base == ".." || base == "/" || base == "" {
		return "", errors.New("the source file has no usable name")
	}
	if err := checkCharacters("source file name", base); err != nil {
		return "", err
	}
	rendered := strings.NewReplacer(
		FilenameToken, base,
		TimestampToken, when.UTC().Format(TimestampLayout),
	).Replace(pattern)
	if err := checkName(rendered + Suffix); err != nil {
		return "", err
	}
	return rendered, nil
}

// Candidate is the name of the collision-th attempt at a backup, counted from
// one. The counter sits before the suffix, which is where 2.0 has always put
// it, so an existing backup library keeps its shape.
func Candidate(pattern string, fileName string, when time.Time, collision int) (string, error) {
	base, err := Render(pattern, fileName, when)
	if err != nil {
		return "", err
	}
	if collision <= 1 {
		return base + Suffix, nil
	}
	return base + "_" + strconv.Itoa(collision) + Suffix, nil
}

// Example is the name the Settings screen previews for one pattern. An invalid
// pattern has no example: the caller shows the rejection instead.
func Example(pattern string) string {
	if Validate(pattern) != nil {
		return ""
	}
	name, err := Candidate(pattern, sampleFileName, sampleTime, 1)
	if err != nil {
		return ""
	}
	return name
}

// Normalise turns a stored or missing pattern into the one actually in effect.
// A configuration written before this setting existed keeps the default name.
func Normalise(pattern string) string {
	if strings.TrimSpace(pattern) == "" {
		return Default
	}
	return pattern
}

// defaultNamePattern recognises the names the fixed 2.0 grammar produced. It is
// how backups created before a custom pattern existed stay visible to
// retention, which otherwise only knows what this application recorded.
func defaultNamePattern(sourceName string) *regexp.Regexp {
	return regexp.MustCompile(
		"^" + regexp.QuoteMeta(sourceName) + `\.([0-9]{14})(?:_[0-9]+)?` +
			regexp.QuoteMeta(Suffix) + "$")
}

// MatchesDefault reports the creation time encoded in a name the fixed 2.0
// grammar produced. A name that does not match is not ours to judge and never
// becomes a retention candidate on the strength of a broad glob.
func MatchesDefault(name string, sourceName string) (time.Time, bool) {
	groups := defaultNamePattern(sourceName).FindStringSubmatch(name)
	if groups == nil {
		return time.Time{}, false
	}
	when, err := time.ParseInLocation(TimestampLayout, groups[1], time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return when, true
}

func checkCharacters(field string, value string) error {
	for _, symbol := range value {
		if symbol < 0x20 || symbol == 0x7f {
			return fmt.Errorf("the backup name %s contains a control character", field)
		}
		if strings.ContainsRune(forbidden, symbol) {
			return fmt.Errorf(
				"the backup name %s contains %q, which a file name cannot carry", field, symbol)
		}
	}
	return nil
}

// checkName applies the rules to a complete rendered name.
func checkName(name string) error {
	if err := checkCharacters("result", name); err != nil {
		return err
	}
	if name == "" || len(name) > maximumNameLength {
		return fmt.Errorf("the backup name must be 1..%d characters long", maximumNameLength)
	}
	if strings.HasPrefix(name, ".") {
		return errors.New("the backup name may not start with a dot")
	}
	if strings.HasSuffix(name, " ") || strings.HasSuffix(name, ".") {
		return errors.New("the backup name may not end with a space or a dot")
	}
	stem, _, _ := strings.Cut(name, ".")
	if windowsDeviceNames[strings.ToLower(stem)] {
		return fmt.Errorf("the backup name %q is reserved on Windows", name)
	}
	return nil
}
