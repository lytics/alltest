/*
Runs all tests in all subdirectories, showing the test stdout. If any of test fails, this
program will exit with a non-zero exit code and print a message.
*/
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/araddon/gou"
)

var (
	verbose  bool
	colorize bool
)

func main() {
	baseDir, err := os.Getwd()
	quitIfErr(err)

	skipDirFlag := flag.String("skip", "trash", "Comma-separated list of directories to skip")
	buildOnlyFlag := flag.Bool("buildOnly", false, "Do \"go build\" instead of \"go test\"")
	shortFlag := flag.Bool("short", false, `Run "go test" with "short" flag`)
	flag.BoolVar(&colorize, "c", true, `colorize output`)
	flag.BoolVar(&verbose, "v", false, `verbose output`)
	raceFlag := flag.Bool("race", false, `Run "go test" with "race" flag`)
	flag.Parse()

	gou.SetLogger(log.New(os.Stderr, "", 0), "debug")
	if colorize {
		gou.SetColorIfTerminal()
	}

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
			gou.Errorf("Couldn't stat directory to skip %s: %s\n", skipDirName, err)
		}
		skipDirStats = append(skipDirStats, stat)
	}

	conf := NewConf(skipDirStats, *buildOnlyFlag, *shortFlag, *raceFlag)
	failedDirs := RunTestsRecursively(baseDir, baseDir, conf)

	if len(failedDirs) > 0 {
		gou.Error("\nFailed directories:")
		for _, dir := range failedDirs {
			gou.Errorf("  %s", dir)
		}
		os.Exit(1)
	} else {
		gou.Info("\nall tests/builds succeeded")
	}
}

func RunTestsRecursively(rootDir, dirName string, conf *Conf) []string {

	if strings.Contains(dirName, "trash") {
		return nil
	}
	// Skip this directory if the user requested that we skip it
	stat, err := os.Stat(dirName)
	quitIfErr(err)
	for _, skipDir := range conf.skipDirs {
		if os.SameFile(stat, skipDir) {
			gou.Debugf("skipping directory %s as requested", dirName)
			return []string{}
		}
	}
	// Skip this directory if the user entered a .alltestignore file
	_, err = os.Stat(path.Join(dirName, ".alltestignore"))
	if err == nil {
		// If err == nil that means we found a file, thus should bail
		gou.Debugf("skipping directory %s as requested due to ignore file", dirName)
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
			failedSubDirs := RunTestsRecursively(rootDir, subDirName, conf)
			failures = append(failures, failedSubDirs...)
		} else if isTestFile(info) {
			anyTestsInDir = true
		} else if isGoFile(info) {
			anyGoSrcsInDir = true
		}
	}

	goRunOpts := []string{"test"}

	// Run "go test" in this directory if it has any tests
	if anyTestsInDir && !conf.buildOnly {
		if conf.short {
			goRunOpts = append(goRunOpts, "-short")
		}
		if conf.race {
			goRunOpts = append(goRunOpts, "-race")
		}
	} else if anyGoSrcsInDir {
		goRunOpts = []string{"build"}
	} else {
		return failures
	}
	err = os.Chdir(dirName)
	quitIfErr(err)
	bytes, err := exec.Command("go", goRunOpts...).Output()
	if len(bytes) > 0 && bytes[len(bytes)-1] == '\n' {
		// lets get rid of last new line at end of this
		bytes = bytes[0 : len(bytes)-2]
	}

	thisDirPath := strings.Replace(dirName, rootDir, "", -1)
	if err != nil {
		if len(bytes) > 0 {
			gou.Errorf(string(bytes))
		}
		gou.Errorf("Failed:   %s", thisDirPath)
		failures = append(failures, thisDirPath)
	} else {
		if verbose && len(bytes) > 0 {
			gou.Debug(string(bytes))
			gou.Infof("Success   %s", thisDirPath)
		}

	}
	return failures
}

type Conf struct {
	skipDirs  []os.FileInfo
	buildOnly bool
	short     bool
	race      bool
}

func NewConf(skipDirs []os.FileInfo, buildOnly, short, race bool) *Conf {
	return &Conf{
		skipDirs:  skipDirs,
		buildOnly: buildOnly,
		short:     short,
		race:      race,
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
		gou.Errorf("Error: %s", err)
		os.Exit(1)
	}
}
