package contract

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const expectedEndpointDefinitionCount = 105

// endpointsDir is the parent directory of this package: the root that holds one
// domain directory per endpoint group.
const endpointsDir = ".."

// Implementation statuses an endpoint contract comment may declare. The status
// is read from the file itself, so implementing an endpoint never requires
// registering it in this test.
const (
	statusContractOnly = "contract definition only"
	statusImplemented  = "implemented"
)

func TestEndpointFilesMatchTheirDefinitions(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(endpointsDir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", endpointsDir, err)
	}

	seenIDs := make(map[string]string, expectedEndpointDefinitionCount)
	definitionCount := 0
	for _, entry := range entries {
		// "swagger" is the standalone local OpenAPI explorer command, not an
		// endpoint group: it holds no endpoint contract file and only calls the
		// getters defined in the domain directories below.
		if !entry.IsDir() || entry.Name() == "contract" || entry.Name() == "itemrouting" ||
			entry.Name() == "swagger" {
			continue
		}

		packageName := entry.Name()
		packageDir := filepath.Join(endpointsDir, packageName)
		files, err := os.ReadDir(packageDir)
		if err != nil {
			t.Fatalf("ReadDir(%s): %v", packageDir, err)
		}
		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".go" || strings.HasSuffix(file.Name(), "_test.go") {
				continue
			}

			path := filepath.Join(packageDir, file.Name())
			name, endpointID := inspectEndpointFile(t, path, packageName)
			definitionCount++

			if want := endpointID + ".go"; file.Name() != want {
				t.Errorf("%s defines %s (%s), want filename %s", path, name, endpointID, want)
			}
			if previous, exists := seenIDs[endpointID]; exists {
				t.Errorf("EndpointID %q is defined by both %s and %s", endpointID, previous, path)
			}
			seenIDs[endpointID] = path
		}
	}

	if definitionCount != expectedEndpointDefinitionCount {
		t.Fatalf("endpoint definition count = %d, want %d", definitionCount, expectedEndpointDefinitionCount)
	}
}

func inspectEndpointFile(t *testing.T, path, packageName string) (string, string) {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", path, err)
	}
	if parsed.Name.Name != packageName {
		t.Errorf("%s package = %s, want %s", path, parsed.Name.Name, packageName)
	}
	status := validateEndpointHeader(t, path, parsed)

	var endpointName string
	var endpointID string
	var definitionName string
	var functions []*ast.FuncDecl
	for _, declaration := range parsed.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			functions = append(functions, function)
			continue
		}

		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 {
				continue
			}
			name := value.Names[0].Name
			switch {
			case general.Tok == token.CONST && strings.HasSuffix(name, "EndpointID"):
				if endpointName != "" || len(value.Values) != 1 {
					t.Fatalf("%s must contain exactly one EndpointID constant", path)
				}
				literal, ok := value.Values[0].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					t.Fatalf("%s EndpointID must be a string literal", path)
				}
				endpointID, err = strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatalf("%s EndpointID: %v", path, err)
				}
				endpointName = strings.TrimSuffix(name, "EndpointID")
			case general.Tok == token.VAR && strings.HasSuffix(name, "Definition"):
				if definitionName != "" {
					t.Fatalf("%s must contain exactly one endpoint definition", path)
				}
				definitionName = strings.TrimSuffix(name, "Definition")
			}
		}
	}

	if endpointName == "" || endpointID == "" {
		t.Fatalf("%s does not define an EndpointID constant", path)
	}
	if definitionName == "" {
		t.Fatalf("%s does not define an endpoint contract", path)
	}
	if endpointName != definitionName {
		t.Fatalf("%s EndpointID belongs to %s but definition belongs to %s", path, endpointName, definitionName)
	}
	validateEndpointFunctions(t, path, status, endpointName, functions)

	return endpointName, endpointID
}

// validateEndpointFunctions enforces that a contract-only file stays free of
// runtime code and that an implemented file exports exactly one plain function
// named after its endpoint. Unexported functions without a receiver are allowed
// in an implemented file as local helpers of that single endpoint.
func validateEndpointFunctions(t *testing.T, path, status, endpointName string, functions []*ast.FuncDecl) {
	t.Helper()

	if status == statusContractOnly {
		for _, function := range functions {
			t.Errorf("%s is a contract definition only but contains runtime function %s", path, function.Name.Name)
		}
		return
	}

	var exported []string
	for _, function := range functions {
		if function.Recv != nil {
			t.Errorf("%s runtime function %s must not be a method", path, function.Name.Name)
			continue
		}
		if function.Name.IsExported() {
			exported = append(exported, function.Name.Name)
		}
	}

	if len(exported) != 1 {
		t.Fatalf("%s is implemented and must export exactly one runtime function %s, found %d: %v",
			path, endpointName, len(exported), exported)
	}
	if exported[0] != endpointName {
		t.Errorf("%s exports runtime function %s, want %s", path, exported[0], endpointName)
	}
}

// validateEndpointHeader checks the mandatory endpoint contract comment and
// returns the implementation status it declares.
func validateEndpointHeader(t *testing.T, path string, parsed *ast.File) string {
	t.Helper()

	if len(parsed.Comments) == 0 || parsed.Comments[0].Pos() > parsed.Package {
		t.Fatalf("%s must start with an endpoint contract comment", path)
	}

	header := parsed.Comments[0].Text()
	requiredFields := []string{
		"Endpoint:",
		"EndpointID:",
		"Purpose:",
		"How it works:",
		"Supported resource types:",
		"Input variables:",
		"GameCatalog variables read:",
		"Implementation status:",
	}
	for _, field := range requiredFields {
		if !strings.Contains(header, field) {
			t.Errorf("%s endpoint contract comment is missing %q", path, field)
		}
	}
	if !strings.Contains(header, "Save variables read:") && !strings.Contains(header, "Save variables processed:") {
		t.Errorf("%s endpoint contract comment must describe save variables read or processed", path)
	}

	return endpointImplementationStatus(t, path, header)
}

// endpointImplementationStatus reads the single "Implementation status:" line of
// the contract comment. The status is the text before the first semicolon; the
// remainder is human-readable justification. An unknown, missing or repeated
// status fails instead of defaulting to either mode.
func endpointImplementationStatus(t *testing.T, path, header string) string {
	t.Helper()

	var declared []string
	for _, line := range strings.Split(header, "\n") {
		remainder, found := strings.CutPrefix(strings.TrimSpace(line), "Implementation status:")
		if !found {
			continue
		}
		status, _, _ := strings.Cut(remainder, ";")
		declared = append(declared, strings.TrimSpace(status))
	}

	if len(declared) != 1 {
		t.Fatalf("%s must declare exactly one implementation status, found %d", path, len(declared))
	}
	switch declared[0] {
	case statusContractOnly, statusImplemented:
		return declared[0]
	default:
		t.Fatalf("%s declares unknown implementation status %q", path, declared[0])
		return ""
	}
}
