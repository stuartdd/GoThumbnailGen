package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type logger struct {
	path         string
	name         []byte
	nameLen      int
	filName      string
	fil          *os.File
	err          error
	logToConsole bool
}

func (lf *logger) close() {
	if lf.fil != nil {
		lf.err = lf.fil.Close()
		lf.fil = nil
	}
}

func (lf *logger) init() {
	// No log file name so ensure log to console works.
	if lf.nameLen == 0 {
		lf.filName = ""
		lf.err = nil // ensure log to console works.
		lf.close()
		return
	}

	newName := lf.formatName(time.Now(), lf.name)
	if newName == lf.filName && lf.fil != nil {
		return
	}

	lf.close()

	lf.filName = newName

	// Create the log file
	fil, err := os.Create(filepath.Join(lf.path, lf.filName))
	if err != nil {
		lf.err = err
		lf.fil = nil // ensure log to console works.
	} else {
		lf.err = nil
		lf.fil = fil
	}
}

func newLogfile(path string, name string, logToConsole bool) *logger {
	lf := &logger{
		path:         path,
		name:         []byte(name),
		nameLen:      len(name),
		filName:      "",
		err:          nil,
		logToConsole: logToConsole,
	}
	lf.init()
	return lf
}

func (l *logger) logLineStdOut(s, prefix string) {
	if prefix != "" {
		os.Stdout.WriteString(prefix)
		os.Stdout.WriteString(" ")
	}
	os.Stdout.WriteString(s)
	os.Stdout.WriteString("\n")
}

func (l *logger) logLineStdErr(s, prefix string) {
	if prefix != "" {
		os.Stdout.WriteString(prefix)
		os.Stdout.WriteString(" ")
	}
	os.Stdout.WriteString(s)
	os.Stdout.WriteString("\n")
}

func (l *logger) logLineFile(s, prefix string) error {
	if l.fil == nil {
		return nil
	}
	if prefix != "" {
		_, err := l.fil.WriteString(prefix)
		if err != nil {
			return err
		}
		_, err = l.fil.WriteString(" ")
		if err != nil {
			return err
		}
	}
	_, err := l.fil.WriteString(s)
	if err != nil {
		return err
	}
	_, err = l.fil.WriteString("\n")
	if err != nil {
		return err
	}
	return nil
}

func (l *logger) Log(s, prefix string) {
	err := l.logLineFile(s, prefix)
	if err != nil {
		l.logLineStdErr(err.Error(), "LOGGER: Failed to write to log file:")
		if !l.logToConsole {
			l.logLineStdOut(s, prefix)
		}
	}
	if l.logToConsole {
		l.logLineStdOut(s, prefix)
	}
}

func (l *logger) formatName(dtIn time.Time, n []byte) string {
	var buff bytes.Buffer
	var c byte
	var e byte
	ln := len(n)
	var v int = 0
	for i := 0; i < ln; i++ {
		c = n[i]
		if c == '%' && (i+1) < ln {
			i++
			e = n[i]
			switch e {
			case 'y':
				buff.WriteString(strconv.Itoa(dtIn.Year()))
			case 'm':
				v = int(dtIn.Month())
				if v < 10 {
					buff.WriteRune('0')
				}
				buff.WriteString(strconv.Itoa(v))
			case 'd':
				v = int(dtIn.Day())
				if v < 10 {
					buff.WriteRune('0')
				}
				buff.WriteString(strconv.Itoa(v))
			case 'H':
				v = int(dtIn.Hour())
				if v < 10 {
					buff.WriteRune('0')
				}
				buff.WriteString(strconv.Itoa(v))
			case 'M':
				v = int(dtIn.Minute())
				if v < 10 {
					buff.WriteRune('0')
				}
				buff.WriteString(strconv.Itoa(v))
			case 'S':
				v = int(dtIn.Second())
				if v < 10 {
					buff.WriteRune('0')
				}
				buff.WriteString(strconv.Itoa(v))
			default:
				buff.WriteByte(c)
				i--
			}
		} else {
			if c > 0 {
				buff.WriteByte(c)
			}
		}
	}
	return buff.String()
}

type Users struct {
	ImageRoot  string
	ImagePaths []string
}

type UserPathInfo struct {
	root  string
	iPath string
	user  string
}

func (p *UserPathInfo) Path() string {
	return filepath.Join(p.root, p.user, p.iPath)
}

type ThumbnailInfo struct {
	ThumbNailsExec       []string
	ThumbNailsExecFile   string
	ThumbNailTimeStamp   string
	ThumbNailFileSuffix  string
	ThumbNailsRoot       string
	ImageExtensions      []string
	ThumbNailsMaxPerFile int
	Verbose              bool
	Resources            map[string]*Users
	LogPath              string
	LogName              string
	LogConsole           bool

	pathList    []*UserPathInfo
	currentPath int
	logger      *logger
}

func NewThumbnailInfo(content []byte, configFileName string, verboseArg bool) *ThumbnailInfo {
	c := string(content)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) > 1 {
			c = strings.ReplaceAll(c, fmt.Sprintf("%%{%s}", pair[0]), pair[1])
		}
	}

	thumbnailInfo := &ThumbnailInfo{
		ThumbNailsExec:       []string{"Undefined"},
		ThumbNailsExecFile:   "",
		ThumbNailsRoot:       "",
		ThumbNailTimeStamp:   "%y_%m_%d_%H_%M_%S_",
		ThumbNailFileSuffix:  ".json",
		ImageExtensions:      make([]string, 0),
		ThumbNailsMaxPerFile: math.MaxInt,
		Verbose:              false,
		Resources:            make(map[string]*Users),
		LogPath:              "",
		LogName:              "",
		LogConsole:           false,
		currentPath:          0,
		pathList:             make([]*UserPathInfo, 0),
		logger:               newLogfile("", "", false),
	}

	err := json.Unmarshal([]byte(c), &thumbnailInfo)
	if err != nil {
		os.Stdout.WriteString(fmt.Sprintf("Failed to understand the config data in the file: %s", configFileName))
		os.Exit(1)
	}
	// Override config if -v arg
	if verboseArg {
		thumbnailInfo.Verbose = true
	}

	fileExists(thumbnailInfo.ThumbNailsRoot, "ThumbNailsRoot", true, 1)
	thumbnailInfo.ThumbNailsRoot = absPath(thumbnailInfo.ThumbNailsRoot)

	if thumbnailInfo.LogName != "" {
		fileExists(thumbnailInfo.LogPath, "LogPath", true, 1)
		thumbnailInfo.LogPath = absPath(thumbnailInfo.LogPath)
		thumbnailInfo.logger = newLogfile(thumbnailInfo.LogPath, thumbnailInfo.LogName, thumbnailInfo.LogConsole)
	} else {
		os.Stdout.WriteString(" ## ")
		os.Stdout.WriteString("Not logging to file")
		os.Stdout.WriteString("\n")
		thumbnailInfo.logger = newLogfile("", "", false)
	}

	if thumbnailInfo.ThumbNailsExecFile == "" {
		os.Stdout.WriteString(fmt.Sprintf("thumbNailsExecFile is not defined in: %s\n", configFileName))
		os.Exit(1)
	}

	thumbnailInfo.ThumbNailsExecFile, err = filepath.Abs(thumbnailInfo.ThumbNailsExecFile)
	if err != nil {
		os.Stdout.WriteString(fmt.Sprintf("thumbNailsExecFile is invalid in: %s. Error: %s\n", configFileName, err.Error()))
		os.Exit(1)
	}

	if verboseArg {
		thumbnailInfo.Verbose = true
	}
	return thumbnailInfo
}

func (tni *ThumbnailInfo) Close() {
	tni.logger.close()
}

func (tni *ThumbnailInfo) Next() *UserPathInfo {
	if len(tni.pathList) == 0 {
		for n, u := range tni.Resources {
			for _, p := range u.ImagePaths {
				tni.pathList = append(tni.pathList, &UserPathInfo{root: absPath(u.ImageRoot), user: n, iPath: p})
			}
		}
		tni.currentPath = -1
	}
	tni.currentPath++
	if tni.currentPath >= len(tni.pathList) {
		return nil
	}
	return tni.pathList[tni.currentPath]
}

func (tni *ThumbnailInfo) ExampleTimeStamp() string {
	fdt := NewFileDateTimeFromTime(time.Now())
	return fdt.Format(tni.ThumbNailTimeStamp)
}

func (tni *ThumbnailInfo) Extensions() []string {
	l := []string{}
	for _, v := range tni.ImageExtensions {
		vl := strings.ToLower(v)
		l = append(l, strings.ToLower(vl))
		if tni.Verbose {
			tni.logger.Log(vl, "Added File Extension:")
		}
	}
	return l
}

func (tni *ThumbnailInfo) String() string {
	var buff bytes.Buffer
	buff.WriteString("\n ## Run At:               ")
	buff.WriteString(time.Now().Format(time.RFC3339))
	buff.WriteString("\n ## ThumbNailsRoot:       ")
	buff.WriteString(tni.ThumbNailsRoot)
	buff.WriteString("\n ## ThumbNailsExecFile:   ")
	buff.WriteString(tni.ThumbNailsExecFile)
	buff.WriteString("\n ## ThumbNailsExec:       ")
	for i, e := range tni.ThumbNailsExec {
		buff.WriteString("\n ##                       ")
		buff.WriteString(pad2(i + 1))
		buff.WriteString(" ")
		buff.WriteString(e)
	}
	buff.WriteString(" --> ")
	buff.WriteString(tni.ThumbNailsRoot)
	buff.WriteString("\n ## ThumbNailTimeStamp:   ")
	buff.WriteString(tni.ThumbNailTimeStamp)
	buff.WriteString(" Example:")
	buff.WriteString(tni.ExampleTimeStamp())
	buff.WriteString("\n ## ThumbNailFileSuffix:  ")
	buff.WriteString(tni.ThumbNailFileSuffix)
	buff.WriteString("\n ## ThumbNailsMaxPerFile: ")
	buff.WriteString(strconv.Itoa(tni.ThumbNailsMaxPerFile))
	buff.WriteString("\n ## Verbose:              ")
	buff.WriteString(fmt.Sprintf("%t", tni.Verbose))
	buff.WriteString("\n ## Log:                  ")
	logJoin := filepath.Join(tni.LogPath, tni.LogName)
	buff.WriteString(logJoin)
	for n, v := range tni.Resources {
		buff.WriteString("\n ## User:")
		buff.WriteString(n)
		buff.WriteString(v.ToUserPath(n))
	}
	return buff.String()
}

func (u *Users) ToUserPath(name string) string {
	var buff bytes.Buffer
	buff.WriteString("\n ##   Root  ")
	absRoot := absPath(u.ImageRoot)
	buff.WriteString(absRoot)
	for _, v := range u.ImagePaths {
		buff.WriteString("\n ##     Path  ")
		fullPath := filepath.Join(absRoot, name, v)
		buff.WriteString(fullPath)
		ok, s := fileExists(fullPath, fmt.Sprintf("User:%s", name), true, 0)
		if !ok {
			buff.WriteString(" --> ")
			buff.WriteString(s)
		}
	}
	return buff.String()
}

func fileExists(filename string, configName string, shouldBeDir bool, abort int) (bool, string) {
	if filename == "" {
		os.Stderr.WriteString(fmt.Sprintf("File Name is Undefined (%s)\n", configName))
		os.Exit(abort)
	}
	fn, err := filepath.Abs(filename)
	if err != nil {
		s := "Invalid Path: " + err.Error()
		if abort > 0 {
			os.Stderr.WriteString(fmt.Sprintf("File (%s) %s %s\n", configName, filename, s))
			os.Exit(abort)
		}
		return false, s
	}
	info, err := os.Stat(fn)
	if os.IsNotExist(err) {
		s := "Does NOT exist"
		if abort > 0 {
			os.Stderr.WriteString(fmt.Sprintf("File (%s)  %s %s\n", configName, fn, s))
			os.Exit(abort)
		}
		return false, s
	}
	if shouldBeDir {
		if info.IsDir() {
			return true, ""
		}
		s := "Should be a directory"
		if abort > 0 {
			os.Stderr.WriteString(fmt.Sprintf("File (%s) %s %s\n", configName, fn, s))
			os.Exit(abort)
		}
		return false, s
	}
	return true, ""
}
