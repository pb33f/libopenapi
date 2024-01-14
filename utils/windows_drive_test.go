package utils

import "testing"

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
