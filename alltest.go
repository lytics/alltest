/*
Runs all tests in all subdirectories, showing the test stdout. If any of test fails, this
program will exit with a non-zero exit code and print a message.
*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/araddon/gou"
)

func main() {
	baseDir, err := os.Getwd()
	quitIfErr(err)

	skipDirFlag := flag.String("skip", "trash", "Comma-separated list of directories to skip")
	buildOnlyFlag := flag.Bool("buildOnly", false, "Do \"go build\" instead of \"go test\"")
	shortFlag := flag.Bool("short", false, `Run "go test" with "short" flag`)
	flag.Parse()

	gou.SetLogger(log.New(os.Stdout, "", log.LstdFlags), "debug")

	skipDirNames := strings.Split(*skipDirFlag, ",")
	skipDirStats := make([]os.FileInfo, 0)
	for _, skipDirName := range skipDirNames {
		if skipDirName == "" {
			continue
		}
		stat, err := os.Stat(skipDirName)
		if skipDirName == "trash" && err != nil {
			continue
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't stat directory to skip %s: %s\n", skipDirName, err.Error())
		}
		skipDirStats = append(skipDirStats, stat)
	}

	conf := NewConf(skipDirStats, *buildOnlyFlag, *shortFlag)
	failedDirs := RunTestsRecursively(baseDir, conf)
	fmt.Printf("\n\n")
	if len(failedDirs) > 0 {
		gou.Log(gou.ERROR, "at least one test or build failed. Failed directories:")
		for _, dir := range failedDirs {
			gou.Logf(gou.ERROR, "  %s", dir)
		}
		os.Exit(1)
	} else {
		print("all tests/builds succeeded\n")
		os.Exit(0)
	}
}

func RunTestsRecursively(dirName string, conf *Conf) []string {

	if strings.Contains(dirName, "trash") {
		return nil
	}
	// Skip this directory if the user requested that we skip it
	stat, err := os.Stat(dirName)
	quitIfErr(err)
	for _, skipDir := range conf.skipDirs {
		if os.SameFile(stat, skipDir) {
			print("skipping directory %s as requested", dirName)
			return []string{}
		}
	}
	// Skip this directory if the user entered a .alltestignore file
	_, err = os.Stat(path.Join(dirName, ".alltestignore"))
	if err == nil {
		// If err == nil that means we found a file, thus should bail
		print("skipping directory %s as requested due to ignore file", dirName)
		return []string{}
	}

	infos, err := ioutil.ReadDir(dirName)
	quitIfErr(err)

	failures := []string{}

	anyTestsInDir := false
	anyGoSrcsInDir := false
	for _, info := range infos {
		if info.IsDir() {
			// Recursively run the tests in each subdirectory
			subDirName := path.Join(dirName, info.Name())
			failedSubDirs := RunTestsRecursively(subDirName, conf)
			failures = append(failures, failedSubDirs...)
		} else if isTestFile(info) {
			anyTestsInDir = true
		} else if isGoFile(info) {
			anyGoSrcsInDir = true
		}
	}

	// Run "go test" in this directory if it has any tests
	if anyTestsInDir && !conf.buildOnly {
		testOpts := []string{"test"}
		if conf.short {
			testOpts = append(testOpts, "-short")
		}
		err = os.Chdir(dirName)
		quitIfErr(err)
		print("Running tests in %s", dirName)
		bytes, err := exec.Command("go", testOpts...).Output()
		os.Stdout.Write(bytes)
		if err != nil {
			gou.Logf(gou.ERROR, "Failed:  %s", dirName)
			failures = append(failures, dirName)
		}
	} else if anyGoSrcsInDir {
		err = os.Chdir(dirName)
		quitIfErr(err)
		print("Building in %s", dirName)
		bytes, err := exec.Command("go", "build").Output()
		os.Stdout.Write(bytes)
		if err != nil {
			failures = append(failures, dirName)
		}
	}

	return failures
}

type Conf struct {
	skipDirs  []os.FileInfo
	buildOnly bool
	short     bool
}

func NewConf(skipDirs []os.FileInfo, buildOnly, short bool) *Conf {
	return &Conf{
		skipDirs:  skipDirs,
		buildOnly: buildOnly,
		short:     short,
	}
}

func isNormalFile(stat os.FileInfo) bool {
	if stat.Mode()&os.ModeType != 0 {
		return false // Not a normal file (pipe, device, directory, etc.)
	}
	return true
}

func isTestFile(stat os.FileInfo) bool {
	return isNormalFile(stat) && strings.HasSuffix(stat.Name(), "_test.go")
}

func isGoFile(stat os.FileInfo) bool {
	return isNormalFile(stat) && strings.HasSuffix(stat.Name(), ".go")
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
