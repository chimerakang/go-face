package face

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"
)

var (
	rec *Recognizer

	idolTests = map[string]string{
		"elkie.jpg":      "Elkie, CLC",
		"chaeyoung.jpg":  "Chaeyoung, Twice",
		"chaeyoung2.jpg": "Chaeyoung, Twice",
		"sejeong.jpg":    "Sejeong, Gugudan",
		"jimin.jpg":      "Jimin, AOA",
		"jimin2.jpg":     "Jimin, AOA",
		"jimin4.jpg":     "Jimin, AOA",
		"meiqi.jpg":      "Mei Qi, WJSN",
		"chaeyeon.jpg":   "Chaeyeon, DIA",
		"chaeyeon3.jpg":  "Chaeyeon, DIA",
		"tzuyu2.jpg":     "Tzuyu, Twice",
		"nayoung.jpg":    "Nayoung, PRISTIN",
		"luda2.jpg":      "Luda, WJSN",
		"joy.jpg":        "Joy, Red Velvet",
	}
)

type Idol struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	BandName string `json:"band_name"`
}

type IdolFace struct {
	Descriptor string `json:"descriptor"`
	IdolID     string `json:"idol_id"`
}

type IdolData struct {
	Idols []Idol     `json:"idols"`
	Faces []IdolFace `json:"faces"`
	byID  map[string]*Idol
}

type TrainData struct {
	samples []Descriptor
	cats    []int32
	labels  map[int]string
}

func getTPath(fname string) string {
	return filepath.Join("testdata", fname)
}

func getIdolData() (idata *IdolData, err error) {
	data, err := ioutil.ReadFile(getTPath("idols.json"))
	if err != nil {
		return
	}
	idata = &IdolData{}
	err = json.Unmarshal(data, idata)
	if err != nil {
		return
	}
	idata.byID = make(map[string]*Idol)
	for i, _ := range idata.Idols {
		idol := &idata.Idols[i]
		idata.byID[idol.ID] = idol
	}
	return
}

func str2descr(s string) Descriptor {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return *(*Descriptor)(unsafe.Pointer(&b[0]))
}

func getTrainData(idata *IdolData) (tdata *TrainData) {
	var samples []Descriptor
	var cats []int32
	labels := make(map[int]string)

	var catID int32
	var prevIdolID string
	catID = -1
	for i, _ := range idata.Faces {
		iface := &idata.Faces[i]
		descriptor := str2descr(iface.Descriptor)
		samples = append(samples, descriptor)
		if iface.IdolID != prevIdolID {
			catID++
			labels[int(catID)] = iface.IdolID
		}
		cats = append(cats, catID)
		prevIdolID = iface.IdolID
	}

	tdata = &TrainData{
		samples: samples,
		cats:    cats,
		labels:  labels,
	}
	return
}

func recognizeFile(fpath string) (catID *int, err error) {
	fd, err := os.Open(fpath)
	if err != nil {
		return
	}
	imgData, err := ioutil.ReadAll(fd)
	if err != nil {
		return
	}
	f, err := rec.RecognizeSingle(imgData)
	if err != nil || f == nil {
		return
	}
	id := rec.Classify(f.Descriptor)
	if id < 0 {
		return
	}
	catID = &id
	return
}

func TestInit(t *testing.T) {
	var err error
	rec, err = NewRecognizer("testdata")
	if err != nil {
		t.Fatalf("Can't init face recognizer: %v", err)
	}
}

func TestNumFaces(t *testing.T) {
	faces, err := rec.RecognizeFile(getTPath("pristin.jpg"))
	if err != nil {
		t.Fatalf("Can't get faces: %v", err)
	}
	numFaces := len(faces)
	if err != nil || numFaces != 10 {
		t.Fatalf("Wrong number of faces: %d", numFaces)
	}
}

func TestIdols(t *testing.T) {
	idata, err := getIdolData()
	if err != nil {
		t.Fatalf("Can't get idol data: %v", err)
	}
	tdata := getTrainData(idata)
	rec.SetSamples(tdata.samples, tdata.cats)

	for fname, expected := range idolTests {
		t.Run(fname, func(t *testing.T) {
			names := strings.Split(expected, ", ")
			expectedIname := names[0]
			expectedBname := names[1]

			catID, err := recognizeFile(getTPath(fname))
			if err != nil {
				t.Fatal(err)
			}
			if catID == nil {
				t.Errorf("%s: expected “%s” but not recognized", fname, expected)
				return
			}
			idolID := tdata.labels[*catID]
			idol := idata.byID[idolID]
			actualIname := idol.Name
			actualBname := idol.BandName

			if expectedIname != actualIname || expectedBname != actualBname {
				actual := fmt.Sprintf("%s, %s", actualIname, actualBname)
				t.Errorf("%s: expected “%s” but got “%s”", fname, expected, actual)
			}
		})
	}
}