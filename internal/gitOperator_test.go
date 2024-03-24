package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestParseDiff(t *testing.T) {
	diffByte, err := os.ReadFile("../test_files/test_diff_1.txt")

	if err != nil {
		t.Fatal(err)
	}

	diff := string(diffByte)

	parsed, err := parseDiff(diff)

	if err != nil {
		t.Fatal(err)
	}

	asJson, err := json.Marshal(parsed)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(asJson))
}

func TestCreatePatch(t *testing.T) {
	diffByte, err := os.ReadFile("../test_files/test_diff_1.txt")

	if err != nil {
		t.Fatal(err)
	}

	diff := string(diffByte)

	parsed, err := parseDiff(diff)

	if err != nil {
		t.Fatal(err)
	}

	patchRequest := FilePatch{
		FilePath: "a/src/components/InputField/InputField.tsx b/src/components/InputField/InputField.tsx",
		HunkIds:  []int{11},
	}

	patch, err := createPatch(parsed, []FilePatch{patchRequest})

	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(patch)
}
