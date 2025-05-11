package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const OfsSOI = 0
const OfsAPP1Marker = 2
const OfsAPP1Size = 4
const OfsExifHeader = 6
const OfsTiffHeader = 12
const OfsMainImageOffset = 16

const TiffRecordSize = 12

type IFDEntry struct {
	IFDAddress   uint32
	TagData      *Tag
	TagFormat    *TagFormat
	ByteCount    uint32
	Value        string
	itemCount    uint32
	dataOrOffset []byte
}

func newIFDEntry(walker *Walker) *IFDEntry {
	// Do these in the right order!
	address := walker.posit
	tagNumber := uint32(walker.BytesToUint(walker.Bytes(2)))

	formatId := TiffFormat(walker.BytesToUint(walker.Bytes(2)))
	itemCount := uint32(walker.BytesToUint(walker.Bytes(4)))

	dataOrOffset := walker.Bytes(4) // Datavalue or Offset to data value

	// Get the tag data from the tagNumber
	tagData := LookUpTagData(tagNumber, walker.tagPath)

	// Get the format from the tagNumber
	tagFmt := LookUpTagFormat(formatId)

	if tagFmt.tiffFormat == FormatUndefined {
		tagFmt = LookUpTagFormat(tagData.validFormats[0])
	}
	byteCount := itemCount * tagFmt.byteLen
	return &IFDEntry{
		IFDAddress:   address,
		TagData:      tagData,
		TagFormat:    tagFmt,
		ByteCount:    byteCount,
		Value:        "",
		itemCount:    itemCount,
		dataOrOffset: dataOrOffset,
	}

}

func (p *IFDEntry) Diagnostics(m string) string {
	len := p.itemCount * p.TagFormat.byteLen
	var loc string
	if len <= 4 {
		loc = fmt.Sprintf("VALUE[%s:%s]", bytesToHex(p.dataOrOffset, ','), p.Value)
	} else {
		loc = fmt.Sprintf("OFFSET[%s] VALUE[%s]", bytesToHex(p.dataOrOffset, ','), p.Value)
	}
	return fmt.Sprintf("IFD:%s TAG[%s:%d:%s] ITEM_COUNT[%d*%d] FORMAT[%s] %s TAG_DESC[%s]", m, p.TagData.TagGroup, p.TagData.TagNum, p.TagData.Name, p.itemCount, p.TagFormat.byteLen, p.TagFormat, loc, p.TagData.LongDesc)
}

func (p *IFDEntry) Output() string {
	return fmt.Sprintf("%s=%s", p.TagData.Name, p.Value)
}

func (p *IFDEntry) isSubDir() bool {
	return p.TagData.IsDir
}

type image struct {
	name       string
	walker     *Walker
	soi        string
	exif       bool
	app1Marker string
	app1Size   uint32 // APP1 data size
	IFDdata    []*IFDEntry
	debug      bool
	selectCB   func(*IFDEntry, *Walker) bool
	logOutput  func(string, string)
}

func NewImage(imagePath string, debug bool, selectCallBack func(*IFDEntry, *Walker) bool, logOutFunc func(string, string)) (img *image, err error) {

	defer func() {
		// Main
		if r := recover(); r != nil {
			img = nil
			err = fmt.Errorf("PANIC:IMG:%s", r)
		}
	}()

	p, err := filepath.Abs(imagePath)
	if err != nil {
		return nil, err
	}
	fil, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer fil.Close()

	walker, err := NewWalker(bufio.NewReader(fil), 1024)
	if err != nil {
		panic(fmt.Sprintf("Failed to read file: %v", err))
	}

	walker.tagPath = MapGroupName["Idf0"]
	image := &image{
		debug:    debug,
		selectCB: selectCallBack,
		name:     imagePath,
		walker:   walker,

		soi:        walker.Pos(OfsSOI).Hex(walker.Bytes(2), ""),
		app1Marker: walker.Pos(OfsAPP1Marker).Hex(walker.Bytes(2), ""),
		// Always Big Endian
		// Size includes the size bytes so sub 2
		app1Size:  uint32(walker.Pos(OfsAPP1Size).bytesToUintBE(walker.Bytes(2)) - 2),
		exif:      walker.Pos(OfsExifHeader).ZstringEquals("Exif"),
		IFDdata:   []*IFDEntry{},
		logOutput: logOutFunc,
	}

	if image.debug {
		image.logOutput(image.Diagnostics("IMG"),"")
		if image.selectCB != nil {
			if !image.selectCB(nil, walker.Clone().Pos(OfsMainImageOffset)) {
				os.Exit(1)
			}
		}
	}

	if debug {
		image.logOutput(walker.LinePrint(0, 12, 3),"")
	}

	if image.soi != "FFD8" {
		return nil, fmt.Errorf("jpeg marker 'FFD8' is missing (Offset %d) found %s. Path:%s", OfsSOI, image.soi, imagePath)
	}
	if image.app1Marker != "FFE1" {
		return nil, fmt.Errorf("jpeg APP1 marker 'FFE1' is missing (Offset %d) found %s. Path:%s", OfsAPP1Marker, image.app1Marker, imagePath)
	}
	if !image.exif {
		return nil, fmt.Errorf("jpeg 'Exif' data marker is missing (Offset %d) found %s. Path:%s", OfsExifHeader, bytesToZString(walker.Pos(OfsExifHeader).Bytes(6)), imagePath)
	}

	tiffHeader := walker.Pos(OfsTiffHeader).Zstring(2)
	if tiffHeader == "II" {
		image.walker.littleE = true
	} else {
		if tiffHeader != "MM" {
			return nil, fmt.Errorf("tiff Header 'II' or 'MM' is missing")
		}
		image.walker.littleE = false
	}
	/*
		The rest of the image data needs to know the littleE setting to work

		Calc the start if the tags Using TIFF Header offset
	*/
	mainTiffDir := image.OffsetToAbs(walker.Pos(OfsMainImageOffset).BytesToUint(walker.Bytes(4)))
	if debug {
		image.logOutput(fmt.Sprintf("DEBUG: MainIFD ABS[0x%x (%d)]", mainTiffDir, mainTiffDir),"")
	}

	var following uint64 = 1
	count := 0
	dirName := "Main IFD"

	for following > 0 {
		image.readDirectory(uint32(mainTiffDir), walker, dirName, 0)
		following = image.walker.BytesToUint(image.walker.Bytes(4))
		mainTiffDir = image.OffsetToAbs(following)
		count++
		dirName = fmt.Sprintf("Dir%d IFD", count)
	}

	image.sortEntries()

	if image.debug {
		for _, ifd := range image.IFDdata {
			image.logOutput(ifd.Output(),"")
		}
	}
	return image, nil
}

func (p *image) OffsetToAbs(offset uint64) uint64 {
	return OfsTiffHeader + offset
}

func (p *image) Diagnostics(m string) string {
	return fmt.Sprintf("DEBUG:%s SOI[%s]  APP1 Mark[%s] APP1 Size[%d] FileLen[%d]Name[%s] LittleE[%t] EXIF[%t]", m, p.soi, p.app1Marker, p.app1Size, p.walker.data.length, p.name, p.walker.littleE, p.IsExif())
}

func (p *image) Output() string {
	var line bytes.Buffer
	for _, o := range p.IFDdata {
		line.WriteString(o.Output())
		line.WriteString("\n")
	}
	return line.String()
}

func (p *image) sortEntries() {
	m := map[string]*IFDEntry{}
	for i, x := range p.IFDdata {
		tag, ok := MapTagsGrouped[x.TagData.TagGroup][x.TagData.TagNum]
		if ok {
			m[tag.Name] = x
		} else {
			m[fmt.Sprintf("x:%4x:%d", x.TagData.TagNum, i)] = x
		}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := make([]*IFDEntry, len(keys))
	for i, s := range keys {
		sorted[i] = m[s]
	}
	p.IFDdata = sorted
}

func (p *image) IsExif() bool {
	return p.exif
}

func (p *image) getValueBytes(ifd *IFDEntry) []byte {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("%s %s", r, ifd.Diagnostics("")))
		}
	}()

	byteCount := ifd.itemCount * ifd.TagFormat.byteLen
	if (byteCount) > 4 {
		// Location is a pointer from the IDFBase
		// Clone the walker so we can use it to get the bytes without effecting the parser
		w := p.walker.Clone()
		pos := p.OffsetToAbs(w.BytesToUint(ifd.dataOrOffset))
		return w.Pos(uint32(pos)).Bytes(byteCount)
	} else {
		// Location is the value
		return ifd.dataOrOffset
	}
}

func (p *image) GetIDFData(ifd *IFDEntry) string {
	var line bytes.Buffer
	bytes := p.getValueBytes(ifd)
	items := int(ifd.itemCount)
	tagFormat := ifd.TagFormat

	if tagFormat.tiffFormat == FormatString {
		return bytesToZString(bytes)
	}

	bytePos := 0
	byteLen := int(tagFormat.byteLen)
	for i := 0; i < items; i++ {
		subBytes := bytes[bytePos : bytePos+byteLen]
		switch tagFormat.tiffFormat {
		case FormatUint8:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToUint(subBytes)))
		case FormatInt8:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToInt(subBytes)))
		case FormatUint16:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToUint(subBytes)))
		case FormatInt16:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToInt(subBytes)))
		case FormatUint32:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToUint(subBytes)))
		case FormatInt32:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToInt(subBytes)))
		case FormatURational:
			n := p.walker.BytesToUint(subBytes[0:4])
			d := p.walker.BytesToUint(subBytes[4:])
			line.WriteString(fmt.Sprintf("%d/%d", n, d))
		case FormatRational:
			n := p.walker.BytesToInt(subBytes[0:4])
			d := p.walker.BytesToInt(subBytes[4:])
			line.WriteString(fmt.Sprintf("%d/%d", n, d))
		default:
			line.WriteString(p.walker.Hex(subBytes, "0x"))
		}
		bytePos = bytePos + byteLen
		if i < (items - 1) {
			line.WriteRune(',')
		}
	}
	return line.String()
}

func (p *image) readDirectory(base uint32, walker *Walker, dirName string, depth int) {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		panic(fmt.Sprintf("PANIC:DIR:%s", r))
	// 	}
	// }()
	dirCount := int(walker.Pos(base).BytesToUint(walker.Bytes(2)))
	if dirCount <= 0 || dirCount > 200 {
		panic(fmt.Sprintf("Image TIFF data count is invalid. Expected[1..200]. Actual=[%d]", dirCount))
	}
	for i := 0; i < dirCount; i++ {
		current := walker.posit
		ne := newIFDEntry(walker)
		if ne.isSubDir() {
			absSubDir := uint32(p.OffsetToAbs(walker.BytesToUint(ne.dataOrOffset)))
			if p.debug {
				wc := walker.Clone()
				dc := wc.Pos(absSubDir).BytesToUint(wc.Bytes(2))
				p.logOutput(fmt.Sprintf("IFD:[%s of %s :%d] %s ENTRIES[%d] DIR[%s] ABS[0x%x (%d)]", pad0(uint32(i+1), 2), pad0(uint32(dirCount), 2), depth, dirName, dc, ne.TagData.Name, absSubDir, absSubDir),"")
			}
			p.readDirectory(absSubDir, walker.CloneWithPath(ne.TagData.TagGroup), ne.TagData.Name, depth+1)
		} else {
			ne.Value = p.GetIDFData(ne)
			if (p.selectCB != nil && p.selectCB(ne, walker.Clone().Pos(current))) || p.selectCB == nil {
				if p.debug {
					p.logOutput(ne.Diagnostics(fmt.Sprintf("[%s of %s :%d] %s ", pad0(uint32(i+1), 2), pad0(uint32(dirCount), 2), depth, dirName)),"")
				}
				p.IFDdata = append(p.IFDdata, ne)
			}
		}
	}
}

func pad0(i uint32, n int) string {
	s := fmt.Sprintf("%d", i)
	if len(s) >= n {
		return s
	}
	return "00000000000000000"[0:n-len(s)] + s
}
