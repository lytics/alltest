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

	failedDirs := RunTestsRecursively(baseDir)
	fmt.Printf("\n\n")
	if len(failedDirs) > 0 {
		print("at least one test failed or couldn't be executed. Failed directories:")
		for _, dir := range failedDirs {
			print("  %s", dir)
		}
	} else {
		print("all tests passed.\n")
	}
}

func RunTestsRecursively(dirName string) []string {
	infos, err := ioutil.ReadDir(dirName)
	quitIfErr(err)

	failures := []string{}

	anyTestsInDir := false
	for _, info := range infos {
		if info.IsDir() {
			// Recursively run the tests in each subdirectory
			subDirName := path.Join(dirName, info.Name())
			failedSubDirs := RunTestsRecursively(subDirName)
			failures = append(failures, failedSubDirs...)
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
			failures = append(failures, dirName)
		}
	}

	return failures
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

func print(fmtStr string, args ...interface{}) {
	fmt.Printf("alltest: "+fmtStr+"\n", args...)
}
