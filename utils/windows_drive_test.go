package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReplaceWindowsDriveWithLinuxPath(t *testing.T) {
	path := `C:\Users\pb33f\go\src\github.com\pb33f\libopenapi\utils\windows_drive_test.go`
	expected := `/Users/pb33f/go/src/github.com/pb33f/libopenapi/utils/windows_drive_test.go`
	result := ReplaceWindowsDriveWithLinuxPath(path)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	path = `/do/not/replace/this/path`
	expected = `/do/not/replace/this/path`
	result = ReplaceWindowsDriveWithLinuxPath(path)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap(t *testing.T) {
	if runtime.GOOS == "windows" {
		pathA := `C:\Users\pb33f`
		pathB := `pb33f\files\thing.yaml`
		expected := `C:\Users\pb33f\files\thing.yaml`
		result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	} else {
		pathA := `/Users/pb33f`
		pathB := `pb33f/files/thing.yaml`
		expected := `/Users/pb33f/files/thing.yaml`
		result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	}
}

func TestCheckPathOverlap_CheckSlash(t *testing.T) {
	pathA := `/Users/pb33f`
	pathB := `Users/pb33f\files\thing.yaml`

	if runtime.GOOS != "windows" {
		expected := `/Users/pb33f/files\thing.yaml`
		result := CheckPathOverlap(pathA, pathB, `\`)
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	} else {
		expected := `\Users\pb33f\files\thing.yaml`
		result := CheckPathOverlap(pathA, pathB, `\`)
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	}
}

func TestCheckPathOverlap_VariationA(t *testing.T) {
	pathA := `/Users/pb33f`
	pathB := `pb33f/files/thing.yaml`
	expected := `/Users/pb33f/files/thing.yaml`
	if runtime.GOOS == "windows" {
		expected = `\Users\pb33f\files\thing.yaml`
	}
	result := CheckPathOverlap(pathA, pathB, `/`)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_VariationB(t *testing.T) {
	pathA := `somewhere/pb33f`
	pathB := `pb33f/files/thing.yaml`
	expected := `somewhere/pb33f/files/thing.yaml`
	if runtime.GOOS == "windows" {
		expected = `somewhere\pb33f\files\thing.yaml`
	}
	result := CheckPathOverlap(pathA, pathB, `/`)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_DedupMixedSeparators(t *testing.T) {
	pathA := filepath.Join("tmp", "spec", "resources", "models")
	pathB := "models/subdir/file.yaml"
	expected := filepath.Join("tmp", "spec", "resources", "models", "subdir", "file.yaml")
	result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_NoOverlap(t *testing.T) {
	pathA := filepath.Join("root", "base")
	pathB := "other/file.yaml"
	expected := filepath.Join("root", "base", "other", "file.yaml")
	result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_EmptySeparatorDefaults(t *testing.T) {
	pathA := filepath.Join("root", "base")
	pathB := "base/file.yaml"
	expected := filepath.Join("root", "base", "file.yaml")
	result := CheckPathOverlap(pathA, pathB, "")
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_OverlapConsumesAll(t *testing.T) {
	pathA := filepath.Join("root", "base")
	pathB := "base"
	expected := filepath.Clean(pathA)
	result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_EmptyInputs(t *testing.T) {
	expected := ""
	result := CheckPathOverlap("", "", string(os.PathSeparator))
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_CaseInsensitiveWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("case-insensitive overlap is Windows-specific")
	}
	pathA := `C:\Users\pb33f\Specs`
	pathB := `specs\models\thing.yaml`
	expected := `C:\Users\pb33f\Specs\models\thing.yaml`
	result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestCheckPathOverlap_DotDotResolution tests that leading ".." segments in pathB
// are resolved against pathA before checking for overlap. This prevents path doubling
// issues when resolving refs like "../components/schemas/X.yaml" from a base path
// like "/path/to/components/schemas".
func TestCheckPathOverlap_DotDotResolution(t *testing.T) {
	// This is the key scenario that caused path doubling bugs:
	// - Base path is inside components/schemas (e.g., from User.yaml)
	// - Ref goes up with ".." then back into components/schemas
	// - Without proper ".." handling, we'd get components/components/schemas/...
	pathA := filepath.Join("tmp", "spec", "components", "schemas")
	pathB := "../components/schemas/Admin.yaml"
	expected := filepath.Join("tmp", "spec", "components", "schemas", "Admin.yaml")
	result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_DotDotMultipleLevels(t *testing.T) {
	// Multiple ".." segments going up multiple levels
	pathA := filepath.Join("tmp", "spec", "components", "schemas", "nested")
	pathB := "../../responses/Problem.yaml"
	expected := filepath.Join("tmp", "spec", "components", "responses", "Problem.yaml")
	result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_DotDotWithOverlap(t *testing.T) {
	// ".." followed by overlap detection
	pathA := filepath.Join("tmp", "spec", "paths")
	pathB := "../components/schemas/User.yaml"
	expected := filepath.Join("tmp", "spec", "components", "schemas", "User.yaml")
	result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_DotDotAbsolutePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		pathA := `C:\spec\components\schemas`
		pathB := `../components/schemas/Admin.yaml`
		expected := `C:\spec\components\schemas\Admin.yaml`
		result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	} else {
		pathA := "/spec/components/schemas"
		pathB := "../components/schemas/Admin.yaml"
		expected := "/spec/components/schemas/Admin.yaml"
		result := CheckPathOverlap(pathA, pathB, string(os.PathSeparator))
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	}
}

// TestCheckPathOverlap_DotDotConsumesAllPathA verifies that when ".." segments
// consume ALL of pathA, the result is just pathB (without the consumed "..").
// This covers line 56-58 in windows_drive.go (len(aParts) == 0 branch).
func TestCheckPathOverlap_DotDotConsumesAllPathA(t *testing.T) {
	// Single segment pathA consumed by ".."
	result := CheckPathOverlap("schemas", "../file.yaml", "/")
	expected := filepath.Join("file.yaml")
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_DotDotConsumesMultipleSegments(t *testing.T) {
	// Two segments pathA, two ".." in pathB - consumes all of pathA
	result := CheckPathOverlap("a/b", "../../file.yaml", "/")
	expected := filepath.Join("file.yaml")
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCheckPathOverlap_DotDotExceedsPathA(t *testing.T) {
	// More ".." than pathA segments - should stop at root
	result := CheckPathOverlap("a", "../../file.yaml", "/")
	expected := filepath.Join("..", "file.yaml")
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestCheckPathOverlap_WindowsDriveLetter tests Windows drive letter detection.
// This covers lines 37-39 in windows_drive.go.
func TestCheckPathOverlap_WindowsDriveLetter(t *testing.T) {
	// Test Windows path parsing with drive letter
	// Note: This uses backslash separator to simulate Windows paths
	pathA := `C:\Users\test`
	pathB := `test\file.yaml`
	result := CheckPathOverlap(pathA, pathB, `\`)
	// Should preserve C: prefix and handle overlap
	if !strings.HasPrefix(result, `C:`) {
		t.Errorf("Expected result to start with C:, got %s", result)
	}
	if !strings.Contains(result, "file.yaml") {
		t.Errorf("Expected result to contain file.yaml, got %s", result)
	}
}
