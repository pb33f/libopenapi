package utils

import (
	"os"
	"path/filepath"
	"runtime"
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
