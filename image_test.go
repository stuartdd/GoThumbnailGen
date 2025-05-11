package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

const tdFileJpeg = "td1.jpg"

var td = []byte{0xff, 0x8, 0xff, 0x4, 0xaf, 0xc6, 0x45, 0x78}

func TestWalkerInt(t *testing.T) {
	createDataFile(t, td, tdFileJpeg)
	walker, err := NewWalker(createReader(t, tdFileJpeg), 100)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(tdFileJpeg)

	walker.SetLittleE(false)

	if walker.BytesToUint(walker.Bytes(1)) != 255 {
		t.Fatal("1st byte != 255")
	}
	w1 := fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "8ff" {
		t.Fatal("int word as hex != 8ff")
	}
	w1 = fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "4af" {
		t.Fatal("int word as hex != 4af")
	}
	w1 = fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "c645" {
		t.Fatal("int word as hex != c645")
	}
	walker.Pos(0)
	walker.SetLittleE(true)
	w1 = fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "8ff" {
		t.Fatal("int word-hi as hex != 8ff")
	}
	w1 = fmt.Sprintf("%x", walker.Pos(6).BytesToUint(walker.Bytes(1)))
	if w1 != "45" {
		t.Fatal("int word-hi as hex != 45")
	}
	w1 = fmt.Sprintf("%x", walker.Pos(6).BytesToUint(walker.Bytes(1)))
	if w1 != "45" {
		t.Fatal("int word-hi as hex != 45")
	}

	walker.Pos(0)
	w1 = fmt.Sprintf("%x", walker.Pos(0).bytesToUintBE(walker.Bytes(4)))
	if w1 != "ff08ff04" {
		t.Fatal("int word-hi as hex != ff08ff04")
	}
	w1 = fmt.Sprintf("%x", walker.Pos(0).bytesToUintLE(walker.Bytes(4)))
	if w1 != "4ff08ff" {
		t.Fatal("int word-hi as hex != 4ff08ff")
	}

	defer func() {
		if r := recover(); r != nil {
			if r != "Failed to extend buffer. Required 1. Current 4, Only able to read 0. Error: EOF" {
				t.Fatalf("SetPos(8) Did not panic with correct message")
			}
		}
	}()
	walker.Pos(8)
	t.Fatal("Should not get here")

}

func TestHex(t *testing.T) {
	b := []byte{}
	for i := 0; i < 256; i++ {
		b = append(b, byte(i))
	}
	createDataFile(t, b, tdFileJpeg)
	walker, err := NewWalker(createReader(t, tdFileJpeg), 100)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(tdFileJpeg)

	res := walker.Hex(walker.Bytes(256), "")
	for i := 0; i < 256; i++ {
		b := res[i*2 : (i*2)+2]
		d, err := strconv.ParseInt("0"+b, 16, 16)
		if err != nil {
			t.Fatalf("Conversion to bin failed for %s", b)
		}
		if i != int(d) {
			t.Fatalf("Conversion to hex failed for %s --> %d at %d", b, d, i)
		}
	}

}

func TestWalkerHex(t *testing.T) {
	createDataFile(t, td, tdFileJpeg)
	walker, err := NewWalker(createReader(t, tdFileJpeg), 100)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(tdFileJpeg)

	if walker.Hex(walker.Bytes(1), "0x") != "0xFF" {
		t.Fatal("1st byte != ff")
	}
	if walker.Hex(walker.Bytes(1), "0x") != "0x08" {
		t.Fatal("byte != 08")
	}
	if walker.Hex(walker.Bytes(2), "0x") != "0xFF04" {
		t.Fatal("word != ff04")
	}
	if walker.Hex(walker.Bytes(1), "") != "AF" {
		t.Fatal("byte != af")
	}
	if walker.Hex(walker.Bytes(2), "") != "C645" {
		t.Fatal("word != c645")
	}
	defer func() {
		if r := recover(); r != nil {
			if r != "Failed to extend buffer. Required 1. Current 8, Only able to read 0. Error: EOF" {
				t.Fatalf("Hex(2) Did not panic with correct message")
			}
		}
	}()
	walker.Hex(walker.Bytes(2), "") // Will take it past end
}

// : FF D8 FF E1 8E 1F 45 78 69 66 00 00 4D 4D 00 2A 00 00 00 08 00 09 88 25 00 04 00 00 00 01 00 00 03 F9 01 10 00 02 00 00 00 08 00 00 00 7A 02 13 00 03 065496 065505 036383 017784 026982 000000 019789 000042 000000 000008 000009 034853 000004 000000 000001 000000 001017 000272 000002 000000 000008 000000 000122 000531 000003
// : ÿ  Ø  ÿ  á      F  x  i  f      M  M    *                %                    ù                        z          FFD8   FFE1   8E1F   4578   6966   0000   4D4D   002A   0000   0008   0009   8825   0004   0000   0001   0000   03F9   0110   0002   0000   0008   0000   007A   0213   0003
var td1 = []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x8E, 0x1F, 0x46, 0x78, 0x69, 0x66, 0x00, 0x00, 0x4D, 0x4D, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x08, 0x00, 0x09, 0x88, 0x25, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x03, 0xF9, 0x01, 0x10, 0x00, 0x02, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x7A, 0x02, 0x13, 0x0D, 0x03}
var td2 = []byte{0xFF, 0xD0, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x84, 0x00, 0x06, 0x04, 0x04, 0x05, 0x04, 0x03, 0x06, 0x05, 0x04, 0x05, 0x06, 0x06, 0x06, 0x07, 0x09, 0x0F, 0x09, 0x09, 0x08, 0x08, 0x09, 0x12, 0x0D, 0x0D, 0x0A}
var td3 = []byte{0xFF, 0xD8, 0xFF, 0xEF, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x84, 0x00, 0x06, 0x04, 0x04, 0x05, 0x04, 0x03, 0x06, 0x05, 0x04, 0x05, 0x06, 0x06, 0x06, 0x07, 0x09, 0x0F, 0x09, 0x09, 0x08, 0x08, 0x09, 0x12, 0x0D, 0x0D, 0x0A}
var td4 = []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x84, 0x00, 0x06, 0x04, 0x04, 0x05, 0x04, 0x03, 0x06, 0x05, 0x04, 0x05, 0x06, 0x06, 0x06, 0x07, 0x09, 0x0F, 0x09, 0x09, 0x08, 0x08, 0x09, 0x12, 0x0D, 0x0D, 0x0A}

func TestWalkerRead2(t *testing.T) {
	createDataFile(t, td1, tdFileJpeg)
	w, err := NewWalker(createReader(t, tdFileJpeg), 2)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		b := w.Advance(1)
		if b != uint32(td1[i]) {
			t.Fatalf("Byte should be %d actual %d", td1[i], b)
		}
	}
	w.Pos(17)
	for i := 17; i < 27; i++ {
		b := w.Advance(1)
		if b != uint32(td1[i]) {
			t.Fatalf("Byte should be %d actual %d", td1[i], b)
		}
	}

}
func TestWalkerRead10(t *testing.T) {
	createDataFile(t, td1, tdFileJpeg)
	w, err := NewWalker(createReader(t, tdFileJpeg), 10)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		b := w.Advance(1)
		if b != uint32(td1[i]) {
			t.Fatalf("Byte should be %d actual %d", td1[i], b)
		}
	}
	w.Pos(17)
	for i := 17; i < 27; i++ {
		b := w.Advance(1)
		if b != uint32(td1[i]) {
			t.Fatalf("Byte should be %d actual %d", td1[i], b)
		}
	}
}

func TestWalkerAdvanceOffEnd(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if r != "Failed to extend buffer. Required 1. Current 50, Only able to read 0. Error: EOF" {
				t.Fatalf("Advance(1) Did not panic with correct message")
			}
		}
	}()
	createDataFile(t, td1, tdFileJpeg)
	w, err := NewWalker(createReader(t, tdFileJpeg), 10)
	if err != nil {
		t.Fatal(err)
	}
	w.Pos(37)
	for i := 37; i < len(td1); i++ {
		b := w.Advance(1)
		if b != uint32(td1[i]) {
			t.Fatalf("Byte should be %d actual %d", td1[i], b)
		}
	}
	// Following should panic EOF
	w.Advance(1)
}

func TestWalkerLastByted(t *testing.T) {
	createDataFile(t, td1, tdFileJpeg)
	for p := 31; p < 49; p++ {
		w, err := NewWalker(createReader(t, tdFileJpeg), 10)
		if err != nil {
			t.Fatal(err)
		}
		w.Pos(uint32(p))
		b := uint32(0)
		for i := p; i < len(td1); i++ {
			b = w.Advance(1)
			if b != uint32(td1[i]) {
				t.Fatalf("Byte should be %d actual %d", td1[i], b)
			}
		}
		if b != 3 {
			t.Fatalf("Last Byte should be 3 actual %d", b)
		}

	}

}

func TestWalkerPosPastEnd(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if r != "Failed to extend buffer. Required 4. Current 37, Only able to read 2." {
				t.Fatalf("w.Pos(51) Did not panic with correct message")
			}
		}
	}()
	createDataFile(t, td1, tdFileJpeg)
	w, err := NewWalker(createReader(t, tdFileJpeg), 10)
	if err != nil {
		t.Fatal(err)
	}
	w.Pos(37)
	// Length of buffer is not 37+10
	//
	// Following should panic EOF. File only has 3 to go we wanted 4
	w.Pos(51)
}

func TestBadExifMarker(t *testing.T) {
	createDataFile(t, td1, tdFileJpeg)
	defer removeDataFile(tdFileJpeg)
	_, err := NewImage(tdFileJpeg, true, nil, logTest)
	if err.Error() != "jpeg 'Exif' data marker is missing (Offset 6) found Fxif. Path:td1.jpg" {
		t.Fatalf("TD1 %s", err.Error())
	}
}

func TestBadSOI(t *testing.T) {
	createDataFile(t, td2, tdFileJpeg)
	defer removeDataFile(tdFileJpeg)
	_, err := NewImage(tdFileJpeg, true, nil, logTest)
	if err.Error() != "jpeg marker 'FFD8' is missing (Offset 0) found FFD0. Path:td1.jpg" {
		t.Fatalf("BadSOI %s", err.Error())
	}
}

func TestBadA001(t *testing.T) {
	createDataFile(t, td3, tdFileJpeg)
	defer removeDataFile(tdFileJpeg)
	_, err := NewImage(tdFileJpeg, true, nil, logTest)
	if err.Error() != "jpeg APP1 marker 'FFE1' is missing (Offset 2) found FFEF. Path:td1.jpg" {
		t.Fatalf("BadA001 %s", err.Error())
	}
}

func TestBadJpg(t *testing.T) {
	createDataFile(t, td4, tdFileJpeg)
	defer removeDataFile(tdFileJpeg)
	_, err := NewImage(tdFileJpeg, true, nil, logTest)
	if err.Error() != "jpeg 'Exif' data marker is missing (Offset 6) found JFIF. Path:td1.jpg" {
		t.Fatalf("%s", err.Error())
	}
}

func createDataFile(t *testing.T, data []byte, fil string) {
	err := os.WriteFile(fil, data, 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func createReader(t *testing.T, fil string) *bufio.Reader {
	f, err := os.Open(fil)
	if err != nil {
		t.Fatal(err)
	}
	return bufio.NewReader(f)
}

func removeDataFile(fil string) {
	os.Remove(fil)
}

const golden string = `DateTimeDigitized=2016:11:06 11:29:18
DateTimeOriginal=2016:11:06 11:29:18
`

func TestImage01(t *testing.T) {
	im, err := NewImage("testdata/test_data_01.ti", false, func(ifd *IFDEntry, w *Walker) bool {
		return strings.Contains(ifd.TagData.Name, "Date")
	}, logTest)
	if err != nil {
		t.Fatal(err)
	}
	if im.Output() != golden {
		t.Fatalf("Output is not same a golden:\n'%s'\n'%s'", im.Output(), golden)
	}
}

func TestImage02(t *testing.T) {
	_, err := NewImage("testdata/test_data_02.ti", false, func(ifd *IFDEntry, w *Walker) bool {
		return strings.Contains(ifd.TagData.Name, "Date")
	}, logTest)
	if err.Error() != "jpeg APP1 marker 'FFE1' is missing (Offset 2) found FFE0. Path:testdata/test_data_02.ti" {
		t.Fatal(err)
	}
}

func TestImage03(t *testing.T) {
	var buff bytes.Buffer
	_, err := NewImage("testdata/test_data_01.ti", false, func(i *IFDEntry, w *Walker) bool {
		buff.WriteString("\n")
		buff.WriteString(i.Diagnostics("TD:"))
		return true
	}, logTest)

	if err != nil {
		t.Fatal(buff.String() + "\n" + err.Error())
	}
	AssertContains(t, buff.String(), []string{
		"TAG[GPSInfo:6:GPSAltitude]",
		"TAG[GPSInfo:5:GPSAltitudeRef]",
		"TAG[Iop:1:InteroperabilityIndex]",
		"TAG[Iop:2:InteroperabilityVersion]",
		"TAG[Idf0:36867:DateTimeOriginal]",
		"VALUE[2016:11:06 11:29:18]",
		"TAG[Idf0:271:Make]",
		"VALUE[LG Electronics]",
	})
}

func logTest(s string, x string) {
	fmt.Println(s)
}

func AssertContains(t *testing.T, s string, c []string) {
	for _, cc := range c {
		if !strings.Contains(s, cc) {
			t.Fatalf("Value does not contain '%s'", cc)
		}
	}
}

// var sampleFromMagick = map[string]string{
// 	"ColorSpace":                        "1",
// 	"ComponentConfiguration":            "0x01,0x02,0x03,0x00",
// 	"DateTimeDigitized":                 "2016:11:06 11:29:18",
// 	"DateTimeOriginal":                  "2016:11:06 11:29:18",
// 	"DigitalZoomRatio":                  "100/100",
// 	"ExifOffset":                        "161",
// 	"ExifVersion":                       "0220",
// 	"ExifImageHeight":                   "2340",
// 	"ExifImageWidth":                    "4160",
// 	"ExposureBiasValue":                 "0/1",
// 	"ExposureTime":                      "1/24",
// 	"Flash":                             "0",
// 	"FlashPixVersion":                   "0100",
// 	"FNumber":                           "240/100",
// 	"FocalLength":                       "3970/1000",
// 	"GPSAltitude":                       "0/1000",
// 	"GPSAltitudeRef":                    "0",
// 	"GPSInfo":                           "1017",
// 	"GPSLatitudeRef":                    "R98",
// 	"InteroperabilityOffset":            "987",
// 	"Make":                              "LG Electronics",
// 	"MeteringMode":                      "2",
// 	"Model":                             "LG-D855",
// 	"Orientation":                       "6",
// 	"PhotographicSensitivity":           "100",
// 	"PixelXDimension":                   "4160",
// 	"PixelYDimension":                   "2340",
// 	"ResolutionUnit":                    "2",
// 	"thumbnail:InteroperabilityIndex":   "R98",
// 	"thumbnail:InteroperabilityVersion": "0100",
// 	"UserComment":                       "    FocusArea=111111111",
// 	"WhiteBalance":                      "0",
// 	"XResolution":                       "72/1",
// 	"YCbCrPositioning":                  "1",
// 	"YResolution":                       "72/1",
// }

/*
ColorSpace=1
ComponentConfiguration=0x01,0x02,0x03,0x00
DateTimeDigitized=2016:11:06 11:29:18
DateTimeOriginal=2016:11:06 11:29:18
DigitalZoomRatio=100/100
ExifImageHeight=2340
ExifImageWidth=4160
ExifVersion=0220
ExposureBiasValue=0/1
ExposureTime=1/24
FNumber=240/100
Flash=0
FlashPixVersion=0100
FocalLength=3970/1000
GPSAltitude=0/1000
GPSAltitudeRef=0
GPSLatitude=0x30,0x31,0x30,0x30
GPSLatitudeRef=R98
ISOSpeedRatings=100
Make=LG Electronics
MeteringMode=2
Model=LG-D855
Orientation=6
ResolutionUnit=2
UserComment=    FocusArea=111111111
WhiteBalance=0
XResolution=72/1
YCbCrPositioning=1
YResolution=72/1

DEBUG:IMG SOI[FFD8]  APP1 Mark[FFE1] APP1 Size[36381] FileLen[1024]Name[../testdata/test_data_01.ti] LittleE[false] EXIF[true] OffsetToMainDir[0] Entries[0]
IFD:[0]GPSIFD TAG[6:GPSAltitude] FORMAT[id:5 bytes:8: type:n/d URational] ITEM_COUNT[1] LOCATION[00000417] VALUE[0/1000] TAG_DESC[Indicates the altitude based on the reference in GPSAltitudeRef.]
IFD:[1]GPSIFD TAG[5:GPSAltitudeRef] FORMAT[id:1 bytes:1: type:Byte Uint8] ITEM_COUNT[1] LOCATION[00000000] VALUE[0] TAG_DESC[Indicates the altitude used as the reference altitude.]
IFD:[1]MainIFD TAG[272:Model] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[8] LOCATION[0000007A] VALUE[LG-D855] TAG_DESC[The scanner model name or number.]
IFD:[2]MainIFD TAG[531:YCbCrPositioning] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00010000] VALUE[1] TAG_DESC[Specifies the positioning of subsampled chrominance components relative to luminance samples.]
IFD:[3]MainIFD TAG[296:ResolutionUnit] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00020000] VALUE[2] TAG_DESC[The unit of measurement for XResolution and YResolution.]
IFD:[4]MainIFD TAG[283:YResolution] FORMAT[id:5 bytes:8: type:n/d URational] ITEM_COUNT[1] LOCATION[00000082] VALUE[72/1] TAG_DESC[The number of pixels per ResolutionUnit in the ImageLength direction.]
IFD:[5]MainIFD TAG[274:Orientation] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00060000] VALUE[6] TAG_DESC[The orientation of the image with respect to the rows and columns.]
IFD:[0]ExifSubIFD TAG[40961:ColorSpace] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00010000] VALUE[1] TAG_DESC[Value is '1'. ]
IFD:[1]ExifSubIFD TAG[36868:DateTimeDigitized] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[20] LOCATION[0000018B] VALUE[2016:11:06 11:29:18] TAG_DESC[Date/Time of image digitized. Usually, it contains the same value of DateTimeOriginal(0x9003). ]
IFD:[2]ExifSubIFD TAG[33437:FNumber] FORMAT[id:5 bytes:8: type:n/d URational] ITEM_COUNT[1] LOCATION[0000019F] VALUE[240/100] TAG_DESC[The actual F-number(F-stop) of lens when the image was taken. ]
IFD:[3]ExifSubIFD TAG[37386:FocalLength] FORMAT[id:5 bytes:8: type:n/d URational] ITEM_COUNT[1] LOCATION[000001A7] VALUE[3970/1000] TAG_DESC[Focal length of lens used to take image. Unit is millimeter. ]
IFD:[4]ExifSubIFD TAG[41987:WhiteBalance] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00000000] VALUE[0] TAG_DESC[Indicates the white balance mode set when the image was shot.]
IFD:[5]ExifSubIFD TAG[40962:ExifImageWidth] FORMAT[id:4 bytes:4: type:Long Uint32] ITEM_COUNT[1] LOCATION[00001040] VALUE[4160] TAG_DESC[Size of main image. ]
IFD:[6]ExifSubIFD TAG[37383:MeteringMode] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00020000] VALUE[2] TAG_DESC[Exposure metering method. '1' means average, '2' center weighted average, '3' spot, '4' multi-spot, '5' multi-segment. ]
IFD:[7]ExifSubIFD TAG[36867:DateTimeOriginal] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[20] LOCATION[000001AF] VALUE[2016:11:06 11:29:18] TAG_DESC[Date/Time of original image taken. This value should not be modified by user program. ]
IFD:[8]ExifSubIFD TAG[37510:UserComment] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[512] LOCATION[000001C3] VALUE[    FocusArea=111111111] TAG_DESC[Stores user comment. ]
IFD:[9]ExifSubIFD TAG[37121:ComponentConfiguration] FORMAT[id:7 bytes:1: type:Undefined] ITEM_COUNT[4] LOCATION[01020300] VALUE[0x01,0x02,0x03,0x00] TAG_DESC[It seems value 0x00,0x01,0x02,0x03 always. ]
IFD:[10]ExifSubIFD TAG[40963:ExifImageHeight] FORMAT[id:4 bytes:4: type:Long Uint32] ITEM_COUNT[1] LOCATION[00000924] VALUE[2340] TAG_DESC[ExifImageHeight ]
IFD:[11]ExifSubIFD TAG[37385:Flash] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00000000] VALUE[0] TAG_DESC['1' means flash was used, '0' means not used. ]
IFD:[12]ExifSubIFD TAG[36864:ExifVersion] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[4] LOCATION[30323230] VALUE[0220] TAG_DESC[Exif version number. Stored as 4bytes of ASCII character (like 0210) ]
IFD:[0]Interoperability TAG[1:GPSLatitudeRef] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[4] LOCATION[52393800] VALUE[R98] TAG_DESC[Indicates whether the latitude is north or south latitude]
IFD:[1]Interoperability TAG[2:GPSLatitude] FORMAT[id:7 bytes:1: type:Undefined] ITEM_COUNT[4] LOCATION[30313030] VALUE[0x30,0x31,0x30,0x30] TAG_DESC[Indicates the latitude]


0000 FFD8 				SOI Start of Image
0002 FFE1 				APP 1 Marker {EXIF}
0004 SSSS 				Size of APP1 (44998 AF C6)
0006 45 78 69 66 00 00 	'Exif00' OfsExifHeader
0012 49 49 2a 00 		'II*0' or 'MM*0' OfsEndianDef
0016 08 00 00 00 		Offset to TIFF (Main Image) OfsMainImageOffset
XXXX 0D 00    			Number of entries (12 bytes each) * BASE

XXXX FFD9 EOI End of Image

Value 				1 				2 				3 				4 				5 					6
Format 				unsigned byte 	ascii strings 	unsigned short 	unsigned long 	unsigned rational 	signed byte
Bytes/component 	1 				1 				2 				4 				8 					1

Value 				7 				8 				9 				10 				11 					12
Format 				undefined 		signed short 	signed long 	signed rational single float 		double float
Bytes/component 	1 				2 				4 				8 				4 					8

magick testImage.jpg -print "%[EXIF:*]\n" info:


0000: 49 49
0002: 2A 00
0004: 08 00 00 00
0008: 02 00 		number of directory entry of IFD0 is '2'

000a: 1A 01 		XResolution(0x011A) Tag
000c: 05 00			Format
000e: 01 00	00 00	Count (Data len = count * Format size)
0010: 26 00 00 00	XResolution Data - Offset

0016: 69 87			ExifOffset offset to Exif SubIFD
0018: 04 00 		Format
001a: 01 00 00 00 	Count (Data len = count * Format size)
001e: 11 02 00 00 	Exif SubIFD starts from address '0x0211'.

0022: 40 00 00 00 	Next IFD

0026: 48 00-00 00 01 00 00 00

If the first part of TIFF data is above, it can read as;

    The first 2bytes are "I I", byte align is 'Intel'.
    Address 0x0004~0x0007 is 0x08000000, IFD0 starts from address '0x0008'
    Address 0x0008~0x0009 is 0x0200, number of directory entry of IFD0 is '2'.
    Address 0x000a~0x000b is 0x1A01, it means this is a XResolution(0x011A) Tag, it contains a horizontal resolution of image.
    Address 0x000c~0x000d is 0x0500, format of this value is unsigned rational(0x0005).
    Address 0x000e~0x0011 is 0x01000000, number of components is '1'. Unsigned rational's data size is 8bytes/components, so total data length is 1x8=8bytes.
    Total data length is larger than 4bytes, so next 4bytes contains an offset to data.
    Address 0x0012~0x0015 is 0x26000000, XResolution data is stored to address 0x0026
    Address 0x0026~0x0029 is 0x48000000, numerator is 72, address 0x002a~0x002d is 0x0100000000, denominator is '1'. So the value of XResoultion is 72/1.
    Address0x0016~0x0017 is 0x6987, next Tag is ExifOffset(0x8769). Its value is an offset to Exif SubIFD
    Data format is 0x0004, unsigned long integer.
    This Tag has one component. Unsigned long integer's data size is 4bytes/components, so total data size is 4bytes.
    Total data size is equal to 4bytes, next 4bytes contains the value of Exif SubIFD offset.
    Address 0x001e~0x0021 is 0x11020000, Exif SubIFD starts from address '0x0211'.
    This is the last directory entry, next 4bytes shows an offset to next IFD.
    Address 0x0022~0x0025 is 0x40000000, next IFD starts from address '0x0040'

*/
