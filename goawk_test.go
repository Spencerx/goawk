// GoAWK tests

package main_test

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/benhoyt/goawk/interp"
	"github.com/benhoyt/goawk/parser"
)

var (
	testsDir   string
	outputDir  string
	writeAWK   bool
	writeGoAWK bool
)

func TestMain(m *testing.M) {
	flag.StringVar(&testsDir, "testsdir", "./tests", "directory with one-true-awk tests")
	flag.StringVar(&outputDir, "outputdir", "./tests/output", "directory for test output")
	flag.BoolVar(&writeAWK, "writeawk", false, "write expected output")
	flag.BoolVar(&writeGoAWK, "writegoawk", true, "write Go AWK output")
	flag.Parse()
	os.Exit(m.Run())
}

func TestAgainstOneTrueAWK(t *testing.T) {
	inputByPrefix := map[string]string{
		"t": "test.data",
		"p": "test.countries",
	}
	nonzeroExits := map[string]bool{
		"t.exit":  true,
		"t.exit1": true,
	}
	// These tests use "for (x in a)", which iterates in an undefined
	// order (according to the spec), so sort lines before comparing.
	sortLines := map[string]bool{
		"p.43": true,
	}

	infos, err := ioutil.ReadDir(testsDir)
	if err != nil {
		t.Fatalf("couldn't read test files: %v", err)
	}
	for _, info := range infos {
		if !strings.HasPrefix(info.Name(), "t.") && !strings.HasPrefix(info.Name(), "p.") {
			continue
		}
		t.Run(info.Name(), func(t *testing.T) {
			srcPath := filepath.Join(testsDir, info.Name())
			inputPath := filepath.Join(testsDir, inputByPrefix[info.Name()[:1]])
			outputPath := filepath.Join(outputDir, info.Name())

			cmd := exec.Command("awk", "-f", srcPath, inputPath)
			expected, err := cmd.Output()
			if err != nil && !nonzeroExits[info.Name()] {
				t.Fatalf("error running awk: %v", err)
			}
			if sortLines[info.Name()] {
				expected = sortedLines(expected)
			}
			if writeAWK {
				err := ioutil.WriteFile(outputPath, expected, 0644)
				if err != nil {
					t.Fatalf("error writing awk output: %v", err)
				}
			}

			output, err := executeGoAWK(srcPath, inputPath)
			if err != nil {
				t.Fatal(err)
			} else {
				if sortLines[info.Name()] {
					output = sortedLines(output)
				}
				if writeGoAWK {
					err := ioutil.WriteFile(outputPath, output, 0644)
					if err != nil {
						t.Fatalf("error writing goawk output: %v", err)
					}
				}
				if string(output) != string(expected) {
					t.Fatalf("output differs, run: git diff %s", outputPath)
				}
			}
		})
	}

	_ = os.Remove("tempbig")
	_ = os.Remove("tempsmall")
}

func executeGoAWK(srcPath, inputPath string) ([]byte, error) {
	src, err := ioutil.ReadFile(srcPath)
	if err != nil {
		return nil, err
	}
	prog, err := parser.ParseProgram(src)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	p := interp.New(buf)
	err = p.ExecBegin(prog)
	if err != nil && err != interp.ErrExit {
		return nil, err
	}
	if err != interp.ErrExit {
		f, errOpen := os.Open(inputPath)
		if errOpen != nil {
			return nil, errOpen
		}
		err = p.ExecFile(prog, inputPath, f)
		f.Close()
		if err != nil && err != interp.ErrExit {
			return nil, err
		}
	}
	err = p.ExecEnd(prog)
	if err != nil && err != interp.ErrExit {
		return nil, err
	}

	return buf.Bytes(), nil
}

func sortedLines(data []byte) []byte {
	trimmed := strings.TrimSuffix(string(data), "\n")
	lines := strings.Split(trimmed, "\n")
	sort.Strings(lines)
	return []byte(strings.Join(lines, "\n") + "\n")
}
