package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var spinnerChars = []rune{'\\', '|', '/', '-', '\\', '|', '/', '-'}

type spinner struct {
	char       int
	every      int
	everyReset int
	prefix     string
}

func NewSpinner(prefix string, every int) *spinner {
	return &spinner{
		char:       0,
		every:      0,
		everyReset: every,
		prefix:     prefix,
	}
}
func (sp *spinner) Reset(prefix string) {
	sp.prefix = prefix
	sp.every = 0
	sp.char = 0
}

func (sp *spinner) Out(pf func(string)) {
	if sp.every <= 0 {
		sp.every = sp.everyReset
		c := spinnerChars[sp.char]
		sp.char++
		if sp.char >= len(spinnerChars) {
			sp.char = 0
		}
		pf(fmt.Sprintf("%c\b", c))
	} else {
		sp.every = sp.every - 1
	}
}

type fileCache struct {
	path  string
	err   error
	files map[string]string
}

func CleanFileCache() *fileCache {
	return &fileCache{
		path:  "",
		files: map[string]string{},
		err:   nil,
	}
}

func NewFileCache(path string, trimPre, trimPost int) *fileCache {
	l := map[string]string{}
	raw, err := os.ReadDir(path)
	if err != nil {
		return &fileCache{
			path:  path,
			files: l,
			err:   err,
		}
	}
	for _, fi := range raw {
		if !fi.IsDir() {
			fn := fi.Name()
			if len(fn) > trimPre+trimPost {
				l[fn[trimPre:len(fn)-trimPost]] = fn
			}
		}
	}
	return &fileCache{
		path:  path,
		files: l,
		err:   nil,
	}
}

func (fc *fileCache) HasFile(fileName string) bool {
	_, ok := fc.files[fileName]
	return ok
}

type TimedProcess struct {
	startTime int64
	endTime   int64
	events    int64
	desc      string
}

func NewTimedProcess(desc string) *TimedProcess {
	tp := &TimedProcess{
		startTime: 0,
		endTime:   0,
		events:    1,
		desc:      desc,
	}
	tp.Start()
	return tp
}

func (t *TimedProcess) Start() {
	st := time.Now().UnixMilli()
	t.startTime = st
	t.endTime = st
}

func (t *TimedProcess) End() {
	st := time.Now().UnixMilli()
	t.endTime = st
}

func (t *TimedProcess) Period() int64 {
	return t.endTime - t.startTime
}

func (t *TimedProcess) Event() {
	t.events++
}

func (t *TimedProcess) String() string {
	p := t.Period()
	s := p / 1000
	ms := p % 1000
	return fmt.Sprintf("%s:(%dms) Events:%d (%dms) min:%s sec:%s ms:%d", t.desc, p, t.events-1, (p % t.events), pad2int64(s/60), pad2int64(s), ms)
}

type FileDateTime struct {
	y, m, d, hh, mm, ss, src int
}

func (dt *FileDateTime) Format(formatString string) string {
	s := strings.ReplaceAll(formatString, "%y", strconv.Itoa(dt.y))
	s = strings.ReplaceAll(s, "%m", pad2(dt.m))
	s = strings.ReplaceAll(s, "%d", pad2(dt.d))
	s = strings.ReplaceAll(s, "%H", pad2(dt.hh))
	s = strings.ReplaceAll(s, "%M", pad2(dt.mm))
	s = strings.ReplaceAll(s, "%S", pad2(dt.ss))
	s = strings.ReplaceAll(s, "%?", pad2(dt.src))
	return s
}

func dirExists(dir string) bool {
	inf, err := os.Stat(dir)
	if err != nil || inf == nil {
		return false
	}
	return inf.IsDir()
}

func pad2int64(i int64) string {
	if i < 10 {
		return "0" + strconv.FormatInt(i, 10)
	}
	return strconv.FormatInt(i, 10)
}

func pad2(i int) string {
	if i < 10 {
		return "0" + strconv.Itoa(i)
	}
	return strconv.Itoa(i)
}

var padString = "                                        "

func padN(i int, n int) string {
	s := strconv.Itoa(i)
	return padString[:n-len(s)] + s
}

func pad4(i int) string {
	if i < 10 {
		return "000" + strconv.Itoa(i)
	}
	if i < 100 {
		return "00" + strconv.Itoa(i)
	}
	if i < 1000 {
		return "0" + strconv.Itoa(i)
	}
	return strconv.Itoa(i)
}

func boolString(b bool) string {
	if b {
		return "YES"
	}
	return "NO "
}

func absPath(p string) string {
	s, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return s
}
