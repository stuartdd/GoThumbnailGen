package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var spinMe = NewSpinner("File: ", 100)

type ExecFile struct {
	tmpl       string
	name       string
	id         int
	open       bool
	fil        *os.File
	inThisFile int
	verbose    bool
	log        func(string, string)
	head       func(*ExecFile) string
	tail       func(*ExecFile) string
}

func NewExecOut(tmpl string, log func(string, string), head func(*ExecFile) string, tail func(*ExecFile) string, verbose bool) (*ExecFile, error) {
	ef := &ExecFile{
		tmpl:       tmpl,
		name:       "",
		id:         0,
		open:       false,
		inThisFile: 0,
		fil:        nil,
		verbose:    verbose,
		log:        log,
		head:       head,
		tail:       tail,
	}

	ef.listExecFiles(func(s string) {
		e := os.Remove(s)
		if e != nil {
			log(fmt.Sprintf("%s. Error%s"+s, e.Error()), "Clean Failed:")
		} else {
			if verbose {
				log(s, "Clean previous exec file:")
			}
		}
	})
	err := ef.Open()
	return ef, err
}

func (ex *ExecFile) listExecFiles(found func(string)) error {
	p := filepath.Dir(ex.tmpl)
	b := filepath.Base(ex.tmpl)
	pn := strings.Index(b, "%n")
	pre := b
	suf := ""
	if pn >= 0 {
		pre = b[:pn]
		suf = b[pn+2:]
	}
	raw, err := os.ReadDir(p)
	if err != nil {
		return err
	}
	for _, fi := range raw {
		if !fi.IsDir() {
			if suf == "" && fi.Name() == b {
				found(filepath.Join(p, fi.Name()))
			} else {
				if strings.HasPrefix(fi.Name(), pre) && strings.HasSuffix(fi.Name(), suf) {
					found(filepath.Join(p, fi.Name()))
				}
			}
		}
	}
	return nil
}

func (ex *ExecFile) Inc() error {
	ex.id++
	return ex.Open()
}

func (ex *ExecFile) WriteString(m string) {
	if ex.open {
		ex.fil.WriteString(m)
		ex.fil.WriteString("\n")
	}
}

func (ex *ExecFile) Open() error {
	n := strings.ReplaceAll(ex.tmpl, "%n", pad4(ex.id))
	if ex.open {
		if n == ex.name {
			return nil
		}
		ex.Close()
	}
	execOut, err := os.Create(n)
	if err != nil {
		return err
	}
	ex.name = n
	ex.open = true
	ex.fil = execOut
	if ex.head != nil {
		ex.WriteString(ex.head(ex))
	}
	ex.log(n, "Exec File Created:")
	return nil
}

func (ex *ExecFile) Close() {
	if ex.open {
		if ex.tail != nil {
			ex.WriteString(ex.tail(ex))
		}
		ex.fil.Close()
		ex.open = false
	}
}

type GroupKey string

type Group struct {
	user   string // The user
	root   string // From resources user imageRoot
	source string // Path to file within root (no filename)
}

func NewGroup(user, root, source string) *Group {
	return &Group{user: user, root: root, source: source}
}

func (g *Group) Key() GroupKey {
	return GroupKey(g.source + "|" + g.user)
}

func (g1 *Group) Equal(g2 *Group) bool {
	if g1.source == g2.source && g1.root == g2.root && g1.user == g2.user {
		return true
	}
	return false
}

func (g *Group) StringToml() string {
	var buff bytes.Buffer
	buff.WriteString("[user=")
	buff.WriteString(g.user)
	buff.WriteString(", root=")
	buff.WriteString(g.root)
	buff.WriteString(", source=")
	buff.WriteString(g.source)
	buff.WriteRune(']')
	return buff.String()
}

type Data struct {
	number       int
	groupData    *Group
	fileName     string // The file name
	tnExists     bool
	tnCreateDone bool
	err          error
}

func NewDataWithError(user, root, source string, err error) *Data {
	return &Data{
		number:       0,
		groupData:    NewGroup(user, root, source),
		fileName:     "",
		tnExists:     false,
		tnCreateDone: false,
		err:          err,
	}
}

func (d *Data) Required() bool {
	return !d.tnExists && !d.tnCreateDone
}

func (d *Data) String() string {
	if d.err != nil {
		return fmt.Sprintf("KEY[%s] %s", d.groupData.Key(), d.err.Error())
	}
	return fmt.Sprintf("group:%s fn:%s tn:%t", d.groupData.StringToml(), d.fileName, d.tnExists)
}

type Dict struct {
	config      *ThumbnailInfo
	tnSuffixLen int
	tnPrefixLen int
	tnPath      string

	list           []*Data             // List of images to be processed
	groups         map[GroupKey]*Group // root,user, and path to reduce duplication in Data
	fileCache      *fileCache          // list of files in current dir. Speed up thumbnail check.
	createdDirs    map[string]bool     // List of create dir paths. To stop multiple create generation
	fileCount      int                 // Number of files found
	tnMissingCount int                 // Number of files without thumbnails
}

func NewDict(config *ThumbnailInfo) *Dict {
	d := &Dict{
		config:      config,
		tnPath:      config.ThumbNailsRoot,
		tnSuffixLen: len(config.ThumbNailFileSuffix),
		tnPrefixLen: len(config.ExampleTimeStamp()),
	}
	d.reset()
	return d
}

func (dict *Dict) reset() {
	dict.groups = map[GroupKey]*Group{}
	dict.list = []*Data{}
	dict.fileCache = CleanFileCache()
	dict.createdDirs = map[string]bool{}
	dict.fileCount = 0
	dict.tnMissingCount = 0
}

func (tni *Dict) LogLine(s string, prefix string) {
	if tni.config != nil {
		tni.config.logger.Log(s, prefix)
	}
}

func (dict *Dict) Populate(timer *TimedProcess, verbose bool) {
	dict.reset()

	extensions := config.Extensions()
	path := dict.config.Next()
	for path != nil {
		scanUserPath(path,
			func(name string) bool {
				// shouldIncludeFile
				if len(config.ImageExtensions) == 0 {
					return true
				}
				for _, v := range extensions {
					if strings.HasSuffix(strings.ToLower(name), v) {
						return true
					}
				}
				return false
			}, // OnFound
			func(d *Data) {
				if d.err == nil {
					dict.fileCount++
					d.tnExists = dict.CheckThumbNailFile(d.fileName, d.groupData)
					if !d.tnExists {
						dict.tnMissingCount++
						d.number = dict.fileCount
						dict.Add(d)
					}
					if verbose {
						config.logger.Log(fmt.Sprintf("%s:%s", pad4(dict.fileCount), d.String()), "")
					} else {
						spinMe.Out(func(s string) {
							os.Stdout.WriteString(s)
						})
					}
					timer.Event()
				} else {
					config.logger.Log(d.String(), "ERROR:")
				}
			})
		path = dict.config.Next()
	}
}

func (d *Dict) Add(data *Data) {
	if data.groupData == nil {
		panic("Tried to add file data wconfig.Verboseith a nil group")
	}
	k := data.groupData.Key()
	g, ok := d.groups[k]
	if ok {
		data.groupData = g
	} else {
		d.groups[k] = data.groupData
	}
	d.list = append(d.list, data)
}

func (d *Dict) GetFileTimeStamp(fileName string, g *Group, logLineFunc func(string, string)) string {
	var dt *FileDateTime
	imagePath := filepath.Join(g.root, g.user, g.source, fileName)
	stat, err := os.Stat(imagePath)
	if err == nil && stat != nil {
		_, err := NewImage(imagePath, false, func(i *IFDEntry, w *Walker) bool {
			if i != nil {
				if i.TagData.Name == "DateTimeOriginal" && dt == nil {
					dt, _ = NewFileDateTimeFromSpec(i.Value, 1)
				}
				if i.TagData.Name == "DateTime" && dt == nil {
					dt, _ = NewFileDateTimeFromSpec(i.Value, 2)
				}
				if i.TagData.Name == "DateTimeDigitized" && dt == nil {
					dt, _ = NewFileDateTimeFromSpec(i.Value, 3)
				}
			}
			return dt != nil
		}, logLineFunc)
		if err != nil {
			logLineFunc(err.Error(),"")
		}
		if dt == nil {
			dt, _ = NewFileDateTimeFromSpec(fileName, 4)
			if dt == nil {
				dt = NewFileDateTimeFromTime(stat.ModTime())
			}
		}
	}
	if dt != nil {
		return dt.Format(d.config.ThumbNailTimeStamp)
	}
	return ""
}

func (d *Dict) CountRequired() int {
	totalRequired := 0
	for _, data := range d.list {
		if data.Required() {
			totalRequired++
		}
	}
	return totalRequired
}

func (d *Dict) CreateMissingTn(timer *TimedProcess, logFn func(string,string), execOut func(string), maxPerFile int) {
	creates := 0
	for _, data := range d.list {
		if data.Required() {
			timer.Event()
			ts := d.GetFileTimeStamp(data.fileName, data.groupData, logFn)
			if ts != "" {
				inFile := filepath.Join(data.groupData.root, data.groupData.user, data.groupData.source, data.fileName)
				outPath := filepath.Join(dict.tnPath, data.groupData.user, data.groupData.source)
				if !dirExists(outPath) {
					_, ok := dict.createdDirs[outPath]
					if !ok {
						execOut(fmt.Sprintf("mkdir -p \"%s\"", outPath))
					}
					dict.createdDirs[outPath] = true
				}
				outFile := filepath.Join(outPath, fmt.Sprintf("%s%s%s", ts, data.fileName, dict.config.ThumbNailFileSuffix))
				ex := ""
				for _, e := range dict.config.ThumbNailsExec {
					ex = strings.ReplaceAll(e, "%in", inFile)
					ex = strings.ReplaceAll(ex, "%out", outFile)
					ex = strings.ReplaceAll(ex, "%count", padN(data.number, 7))
					execOut(ex)
				}
				data.tnCreateDone = true
				creates++
				if creates >= maxPerFile {
					return
				}
			} else {
				logFn(fmt.Sprintf("Time stamp could not be derived. File:%s", data.fileName),"")
			}
		}
	}
}

func (d *Dict) CheckThumbNailFile(fileName string, g *Group) bool {
	tnPath := filepath.Join(d.tnPath, g.user, g.source)
	if d.fileCache.path != d.tnPath {
		d.fileCache = NewFileCache(tnPath, d.tnPrefixLen, d.tnSuffixLen)
	}
	return d.fileCache.HasFile(fileName)
}

func (d *Dict) LogGroups(prefix string) {
	for _, v := range d.groups {
		d.LogLine(fmt.Sprintf("%s %s", prefix, v.StringToml()),"")
	}
}

func (d *Dict) LogDict() {
	if len(d.list) > 0 {
		group := NewGroup("", "", "")
		for _, data := range d.list {
			if !data.groupData.Equal(group) {
				group = data.groupData
				d.LogLine(group.StringToml(),"")
			}
			d.LogLine(fmt.Sprintf("image=%s has thumbnail %s", data.fileName, boolString(data.tnExists)),"")
		}
	}
}

func scanUserPath(upi *UserPathInfo, shouldIncludeFile func(string) bool, onFound func(*Data)) {

	pathTrim := len(upi.root) + 1 + len(upi.user) + 1
	filepath.WalkDir(upi.Path(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			onFound(NewDataWithError(upi.user, upi.root, path, err))
			return err
		} else {
			if d == nil {
				err := fmt.Errorf("%s File info is nil", path)
				onFound(NewDataWithError(upi.user, upi.root, path, err))
				return err
			}
		}
		if !d.IsDir() {
			add := true
			if shouldIncludeFile != nil {
				add = shouldIncludeFile(d.Name())
			}
			if add {
				onFound(&Data{
					groupData: NewGroup(upi.user, upi.root, path[pathTrim:len(path)-len(d.Name())-1]),
					fileName:  d.Name(),
					tnExists:  false, // Will be updated by onFound Method
					err:       nil,
				})
			}
		}
		return nil
	})
}

func NewFileDateTimeFromSpec(spec string, src int) (*FileDateTime, error) {
	spec1 := []byte(spec)
	spec2 := make([]byte, 18)
	specPos := 0
	for _, c := range spec1 {
		if c >= '0' && c <= '9' {
			spec2[specPos] = c
			specPos++
			if specPos > 17 {
				return nil, fmt.Errorf("character buffer overrun")
			}
		}
	}
	specPos = 0
	y, spespecPos := readIntFromSpec(spec2, 0, 4)
	if y < 1970 {
		return nil, fmt.Errorf("year '%d' before 1970", y)
	}
	if y > 2100 {
		return nil, fmt.Errorf("year '%d' after 2070", y)
	}
	m, spespecPos := readIntFromSpec(spec2, spespecPos, 2)
	if m < 1 {
		return nil, fmt.Errorf("month '%d' is 0", m)
	}
	if m > 12 {
		return nil, fmt.Errorf("month '%d' above 12", m)
	}
	d, spespecPos := readIntFromSpec(spec2, spespecPos, 2)
	if d < 1 {
		return nil, fmt.Errorf("day Of Month '%d' is 0", d)
	}
	if m > 31 {
		return nil, fmt.Errorf("day Of Month '%d' above 31", d)
	}
	hh, spespecPos := readIntFromSpec(spec2, spespecPos, 2)
	if hh > 23 {
		return nil, fmt.Errorf("hour '%d' is above 23", hh)
	}
	mm, spespecPos := readIntFromSpec(spec2, spespecPos, 2)
	if mm > 59 {
		return nil, fmt.Errorf("min '%d' is above 59", mm)
	}
	ss, _ := readIntFromSpec(spec2, spespecPos, 2)
	if ss > 59 {
		return nil, fmt.Errorf("seconds '%d' is above 59", ss)
	}
	return &FileDateTime{y: y, m: m, d: d, hh: hh, mm: mm, ss: ss, src: src}, nil
}

func readIntFromSpec(spec []byte, from, len int) (int, int) {
	acc := 0
	for i := from; i < (from + len); i++ {
		acc = acc * 10
		si := int(spec[i])
		if si >= '0' && si <= '9' {
			acc = acc + (int(spec[i] - '0'))
		}
	}
	return acc, from + len
}

func NewFileDateTimeFromTime(t time.Time) *FileDateTime {
	return &FileDateTime{y: t.Year(), m: int(t.Month()), d: t.Day(), hh: t.Hour(), mm: t.Minute(), ss: t.Second(), src: 0}
}
