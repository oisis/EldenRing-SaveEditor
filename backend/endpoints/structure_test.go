package endpoints_test

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

const expectedEndpointDefinitionCount = 100

func TestEndpointFilesMatchTheirDefinitions(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	seenIDs := make(map[string]string, expectedEndpointDefinitionCount)
	definitionCount := 0
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "contract" || entry.Name() == "itemrouting" {
			continue
		}

		packageName := entry.Name()
		files, err := os.ReadDir(packageName)
		if err != nil {
			t.Fatalf("ReadDir(%s): %v", packageName, err)
		}
		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".go" || strings.HasSuffix(file.Name(), "_test.go") {
				continue
			}

			path := filepath.Join(packageName, file.Name())
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
	validateEndpointHeader(t, path, parsed)

	var endpointName string
	var endpointID string
	var definitionName string
	for _, declaration := range parsed.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			t.Errorf("%s contains runtime function %s; contract files must not pretend to implement handlers", path, function.Name.Name)
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
	return endpointName, endpointID
}

func validateEndpointHeader(t *testing.T, path string, parsed *ast.File) {
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
}
