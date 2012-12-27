/*
Runs all tests in all subdirectories, showing the test stdout. If any of test fails, this
program will exit with a non-zero exit code and print a message.
*/
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

func main() {
	baseDir, err := os.Getwd()
	quitIfErr(err)

	if !RunTestsRecursively(baseDir) {
		fmt.Printf("\n\n!!! At least one test failed or couldn't be executed\n")
	}
}

func RunTestsRecursively(dirName string) bool {
	infos, err := ioutil.ReadDir(dirName)
	quitIfErr(err)

	anyFailures := false

	anyTestsInDir := false
	for _, info := range infos {
		if info.IsDir() {
			// Recursively run the tests in each subdirectory
			subDirName := path.Join(dirName, info.Name())
			if !RunTestsRecursively(subDirName) {
				anyFailures = true
			}
		} else if IsTestFile(info) {
			anyTestsInDir = true
		}
	}

	// Run "go test" in this directory if it has any tests
	if anyTestsInDir {
		err = os.Chdir(dirName)
		quitIfErr(err)
		bytes, err := exec.Command("go", "test").Output()
		os.Stdout.Write(bytes)
		if err != nil {
			anyFailures = true
		}
	}

	return !anyFailures
}

func IsTestFile(stat os.FileInfo) bool {
	if stat.Mode()&os.ModeType != 0 {
		return false // Not a normal file (pipe, device, directory, etc.)
	}

	if !strings.HasSuffix(stat.Name(), "_test.go") {
		return false
	}

	return true
}

func quitIfErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}
