package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"
)

var config *ThumbnailInfo
var dict *Dict
var execOut *ExecFile
var verboseArg = false

func main() {
	if len(os.Args) < 2 {
		os.Stderr.WriteString("Requires configFileName")
		os.Stderr.WriteString("\n")
		os.Exit(1)
	}
	for _, a := range os.Args {
		lca := strings.ToLower(a)
		if lca == "-v" {
			verboseArg = true
			os.Stdout.WriteString(" ## Verbose flag found:")
			os.Stdout.WriteString("\n")
		}
	}

	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to read config: %s\n", os.Args))
		os.Exit(1)
	}

	config = NewThumbnailInfo(content, os.Args[1], verboseArg)

	if verboseArg {
		config.Verbose = true
	}

	if config.Verbose {
		config.logger.Log(config.String(), "Config Data:")
	}
	defer config.Close()

	execOut, err = NewExecOut(config.ThumbNailsExecFile, config.logger.Log, head, tail, config.Verbose)
	if err != nil {
		config.logger.Log(fmt.Sprintf("File:%s Error:%s", config.ThumbNailsExecFile, err), "Failed to open exec file:")
		os.Exit(1)
	}
	defer execOut.Close()
	dict = NewDict(config)

	timer := NewTimedProcess("Time to Populate")
	dict.Populate(timer, config.Verbose)
	timer.End()

	config.logger.Log(timer.String(), "")
	if config.Verbose {
		dict.LogGroups("Group")
		dict.LogDict()
	}
	timer = NewTimedProcess("Time Create Script(s)")
	todo := dict.CountRequired()
	for todo > 0 {
		dict.CreateMissingTn(timer, config.logger.Log, execLog, config.ThumbNailsMaxPerFile)
		todoPrev := todo
		todo = dict.CountRequired()
		execOut.inThisFile = todoPrev - todo
		if todo > 0 {
			err := execOut.Inc()
			if err != nil {
				config.logger.Log(fmt.Sprintf("Failed to open/create exec file: %s Errro:%s", config.ThumbNailsExecFile, err), "")
				os.Exit(1)
			}
		}
	}
	timer.End()
	config.logger.Log(timer.String(),"")
	execOut.listExecFiles(func(s string) {
		config.logger.Log("Exec CHMOD:"+s, "")
		os.Chmod(s, 0775)
	})
}

func head(ex *ExecFile) string {
	var buf bytes.Buffer
	buf.WriteString("#!/bin/bash\n\n")
	buf.WriteString(fmt.Sprintf("# Generated At: %s\n", time.Now().Format(time.DateTime)))
	buf.WriteString(fmt.Sprintf("# Args: %s\n", argsToString()))
	buf.WriteString(fmt.Sprintf("# ID: %d\n", ex.id))
	buf.WriteString("# HEAD ------------------------------------\n\n")
	buf.WriteString("createdCount=0\n")
	return buf.String()
}

func tail(ex *ExecFile) string {
	var buf bytes.Buffer
	buf.WriteString("# TAIL ------------------------------------\n\n")
	buf.WriteString(fmt.Sprintf("echo \"Required: %d Created $createdCount\"\n", ex.inThisFile))
	buf.WriteString(fmt.Sprintf("if [ \"%d\" -ne \"$createdCount\" ]; then\n", ex.inThisFile))
	buf.WriteString("  exit 1\n")
	buf.WriteString("  echo \"Not all files were created\"\n")
	buf.WriteString("else\n")
	buf.WriteString("  echo \"All files were created OK\"\n")
	buf.WriteString("fi\n")
	return buf.String()
}

func execLog(s string) {
	if config.Verbose {
		config.logger.Log(s, "Exec:")
	}
	execOut.WriteString(s)
}

func argsToString() string {
	var buf bytes.Buffer
	for _, v := range os.Args {
		buf.WriteString(v)
		buf.WriteRune(' ')
	}
	return buf.String()
}
