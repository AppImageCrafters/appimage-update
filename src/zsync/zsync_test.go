package zsync

import (
	"appimage-update/src/zsync/control"
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

var dataDir string = "/tmp/appimage-update"
var serverUrl string = ""

func TestSync1stChunkChanged(t *testing.T) {
	zsyncControl, _ := getControl()
	zsyncControl.URL = serverUrl + "file"

	local, err := os.Open(dataDir + "/1st_chunk_changed")
	if err != nil {
		return
	}
	defer local.Close()

	output := bytes.Buffer{}

	err = Sync(local, &output, *zsyncControl)
	if err != nil {
		t.Fatal(err)
	}

	expected, _ := ioutil.ReadFile(dataDir + "/file")
	assert.Equal(t, expected, output.Bytes())
}

func TestSync2ndChunkChanged(t *testing.T) {
	zsyncControl, _ := getControl()
	zsyncControl.URL = serverUrl + "file"

	local, err := os.Open(dataDir + "/2nd_chunk_changed")
	if err != nil {
		return
	}
	defer local.Close()

	output := bytes.Buffer{}

	err = Sync(local, &output, *zsyncControl)
	if err != nil {
		t.Fatal(err)
	}

	expected, _ := ioutil.ReadFile(dataDir + "/file")
	assert.Equal(t, expected, output.Bytes())
}

func TestSync3rdChunkChanged(t *testing.T) {
	zsyncControl, _ := getControl()
	zsyncControl.URL = serverUrl + "file"

	local, err := os.Open(dataDir + "/3rd_chunk_changed")
	if err != nil {
		return
	}
	defer local.Close()

	output := bytes.Buffer{}

	err = Sync(local, &output, *zsyncControl)
	if err != nil {
		t.Fatal(err)
	}

	expected, _ := ioutil.ReadFile(dataDir + "/file")
	assert.Equal(t, expected, output.Bytes())
}

func getControl() (zsyncControl *control.Control, err error) {
	data, err := ioutil.ReadFile(dataDir + "/file.zsync")
	if err != nil {
		return nil, err
	}
	zsyncControl, err = control.ParseControl(data)
	if err != nil {
		return nil, err
	}

	return zsyncControl, nil
}

func teardown() {
	os.RemoveAll(dataDir)
}

func setup() {
	dataDir := generateTestDataDir()
	serve(dataDir)
}
func serve(dataDir string) {
	srv := &http.Server{Addr: ":8080"}
	serverUrl = "http://localhost:8080/"

	http.Handle("/", http.FileServer(http.Dir(dataDir)))
	go srv.ListenAndServe()
}

func generateTestDataDir() string {
	err := os.MkdirAll(dataDir, 0777)
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())

	_ = generateFile([]byte("0123456789"), err, dataDir+"/file")
	makeZsyncFile(dataDir+"/file", err)

	_ = generateFile([]byte("x123456789"), err, dataDir+"/1st_chunk_changed")
	_ = generateFile([]byte("0x23456789"), err, dataDir+"/2nd_chunk_changed")
	_ = generateFile([]byte("01x3456789"), err, dataDir+"/3rd_chunk_changed")

	return dataDir
}

func generateFile(letterRunes []byte, err error, filePath string) error {
	baseString := make([]byte, 2048*2+60)
	for i := range baseString {
		baseString[i] = letterRunes[(i/2048)%9]
	}

	err = writeStringToFile(filePath, baseString)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func writeStringToFile(baseFilePath string, baseString []byte) error {
	err := ioutil.WriteFile(baseFilePath, baseString, 0666)
	if err != nil {
		fmt.Print(err)
	}
	return err
}

func makeZsyncFile(baseFileName string, err error) string {
	cmd := exec.Command("zsyncmake", baseFileName)
	cmd.Dir = filepath.Dir(baseFileName)
	err = cmd.Run()
	if err != nil {
		fmt.Print(err)
	}

	return baseFileName + ".zsync"
}
