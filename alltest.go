/*
Runs all tests in all subdirectories, showing the test stdout. If any of test fails, this
program will exit with a non-zero exit code and print a message.
*/
package main

import (
	"flag"
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

	skipDirFlag := flag.String("skip", "", "Comma-separated list of directories to skip")
	flag.Parse()

	skipDirNames := strings.Split(*skipDirFlag, ",")
	skipDirStats := make([]os.FileInfo, 0)
	for _, skipDirName := range skipDirNames {
		if skipDirName == "" {
			continue
		}
		stat, err := os.Stat(skipDirName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't stat directory to skip %s: %s\n", skipDirName,
				err.Error())
			os.Exit(1)
		}
		skipDirStats = append(skipDirStats, stat)
	}

	failedDirs := RunTestsRecursively(baseDir, skipDirStats)
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

func RunTestsRecursively(dirName string, skipDirs []os.FileInfo) []string {
	// Skip this directory if the user requested that we skip it
	stat, err := os.Stat(dirName)
	quitIfErr(err)
	for _, skipDir := range skipDirs {
		if os.SameFile(stat, skipDir) {
			print("skipping directory %s as requested", dirName)
			return []string{}
		}
	}

	infos, err := ioutil.ReadDir(dirName)
	quitIfErr(err)

	failures := []string{}

	anyTestsInDir := false
	for _, info := range infos {
		if info.IsDir() {
			// Recursively run the tests in each subdirectory
			subDirName := path.Join(dirName, info.Name())
			failedSubDirs := RunTestsRecursively(subDirName, skipDirs)
			failures = append(failures, failedSubDirs...)
		} else if IsTestFile(info) {
			anyTestsInDir = true
		}
	}

	// Run "go test" in this directory if it has any tests
	if anyTestsInDir {
		err = os.Chdir(dirName)
		quitIfErr(err)
		print("Running tests in %s", dirName)
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
