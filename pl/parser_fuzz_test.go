package pl

// Here the testing of parser just verify its binary result, ie whether it passes
// or failed. W.R.T the actual correctness, it is verified by the evaluation
// part for simplicity.

import (
	"fmt"
	"os"
	pa "path"
	"path/filepath"
	"runtime"
	"testing"
)

func verifyParser(code string) bool {
	p := newParser(code, nil)
	_, err := p.parse()
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		fmt.Printf("code:\n%s\n", code)
		return false
	}
	return true
}

var testPath = "assets/test"

func getTestPath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir, _ := filepath.Split(filename)
	dir = dir[:len(dir)-1]
	return filepath.Join(
		filepath.Dir(dir),
		testPath,
	)
}

func setC(
	t *testing.F,
) error {
	tpath := getTestPath()

	fs, err := os.ReadDir(tpath)
	if err != nil {
		return err
	}

	for _, f := range fs {
		if f.IsDir() {
			continue
		}
		if filepath.Ext(f.Name()) != ".pl" {
			continue
		}
		fpath := pa.Join(tpath, f.Name())
		data, err := os.ReadFile(fpath)
		if err != nil {
			return err
		}

		t.Add(string(data))
	}
	return nil
}

func FuzzParser(f *testing.F) {
	if err := setC(f); err != nil {
		f.Errorf("err: %s", err.Error())
	}

	f.Fuzz(func(t *testing.T, data string) {
		t.Logf("code: %s", data)
		verifyParser(data)
	})
}
