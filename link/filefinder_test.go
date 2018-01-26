package link

import (
	"context"
	"github.com/GeneralElectric/TrueConnect-Link/trueconnect"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestCancelAddFound(tests *testing.T) {
	testfile1, err := CreateTestFile("./TestCancelAddFoundFile/testfile1.abc")
	if err != nil {
		tests.Fatal(err.Error())
	}

	target := Target{
		Location:  testfile1,
		Recursive: false}

	client := linkClient{}
	currentContext, cancelMethod := context.WithCancel(context.Background())
	client.currentContext = currentContext
	client.statusRecorder = createStatusRecorder(currentContext)

	defer os.RemoveAll("./TestFindLocationsEqualFile")

	foundFiles := make(chan foundFile)
	waitgroup := sync.WaitGroup{}
	waitgroup.Add(1)
	go func() {
		err := client.findFiles(target, &foundFiles)
		if err == nil || err.Error() != "Terminating" {
			tests.Fatal("Unexpected err return from findfiles: " + err.Error())
		}
		close(foundFiles)
		waitgroup.Done()
	}()

	cancelMethod()
	waitgroup.Wait() // panic with deadlock if fails

}

func TestFindLocationsEqualFile(tests *testing.T) {

	testfile1, err := CreateTestFile("./TestFindLocationsEqualFile/testfile1.abc")
	if err != nil {
		tests.Fatal(err.Error())
	}

	defer os.RemoveAll("./TestFindLocationsEqualFile")

	_, err = CreateTestFile("./TestFindLocationsEqualFile/testfile2.abc")
	if err != nil {
		tests.Fatal(err.Error())
	}
	defer os.RemoveAll("./TestFindLocationsEqualFile")

	target := Target{
		Location:  testfile1,
		Match:     "^.*.abc$",
		Recursive: false,
	}

	foundFiles := make(chan foundFile, 10)
	client := linkClient{}
	currentContext, cancelFunction := context.WithCancel(context.Background())
	client.currentContext = currentContext
	client.statusRecorder = createStatusRecorder(currentContext)
	err = client.findFiles(target, &foundFiles)
	if err != nil && err != io.EOF {
		tests.Fatal("Unexpected err return from findfiles: " + err.Error())
	}
	close(foundFiles)
	counter := 0
	for foundFile := range foundFiles {
		if foundFile.uri != testfile1 {
			tests.Fatal("Found: " + foundFile.uri + " Expected: " + testfile1)
		}
		counter++
	}

	if counter != 1 {
		tests.Fatal("not all expected files were found")
	}
	cancelFunction()
}

func TestNotFindNonrecursive(tests *testing.T) {

	testfile1, err := CreateTestFile("./TestNotFindNonrecursive/testfile1.abc")
	if err != nil {
		tests.Fatal(err.Error())
	}
	defer os.RemoveAll("./TestNotFindNonrecursive")

	_, err = CreateTestFile("./TestNotFindNonrecursive/testfolder2/testfile2.abc")
	if err != nil {
		tests.Fatal(err.Error())
	}
	defer os.RemoveAll("./TestNotFindNonrecursive")

	target := Target{
		Location:  filepath.Dir(testfile1),
		Match:     "^.*.abc$",
		Recursive: false,
	}

	expected := []foundFile{
		{target: &target, uri: testfile1},
	}

	foundFiles := make(chan foundFile, 10)
	client := linkClient{}
	currentContext, cancelFunction := context.WithCancel(context.Background())
	client.currentContext = currentContext
	client.statusRecorder = createStatusRecorder(currentContext)
	err = client.findFiles(target, &foundFiles)
	if err != nil && err != io.EOF {
		tests.Fatal("Unexpected err return from findfiles: " + err.Error())
	}
	close(foundFiles)
	counter := 0
	for foundFile := range foundFiles {
		if foundFile.uri != expected[counter].uri {
			tests.Fatal("Found: " + foundFile.uri + " Expected: " + expected[counter].uri)
		}
		counter++
	}

	if counter != len(expected) {
		tests.Fatal("not all expected files were found")
	}
	cancelFunction()
}

func TestFindRecursive(tests *testing.T) {

	testfile1, err := CreateTestFile("./TestFindRecursive/testfile1.abc")
	if err != nil {
		tests.Fatal(err.Error())
	}
	defer os.RemoveAll("./TestFindRecursive")

	testfile2, err := CreateTestFile("./TestFindRecursive/testfolder2/testfile2.abc")
	if err != nil {
		tests.Fatal(err.Error())
	}
	defer os.RemoveAll("./TestFindRecursive")

	target := Target{
		Location:  filepath.Dir(testfile1),
		Match:     "^.*.abc$",
		Recursive: true,
	}

	expected := []foundFile{
		{target: &target, uri: testfile1},
		{target: &target, uri: testfile2},
	}

	foundFiles := make(chan foundFile, 10)
	client := linkClient{}
	currentContext, cancelFunction := context.WithCancel(context.Background())
	client.currentContext = currentContext
	client.statusRecorder = createStatusRecorder(currentContext)
	err = client.findFiles(target, &foundFiles)
	if err != nil && err != io.EOF {
		tests.Fatal("Unexpected err return from findfiles: " + err.Error())
	}
	close(foundFiles)

	counter := 0
	for foundFile := range foundFiles {
		if foundFile.uri != expected[counter].uri {
			tests.Fatal("Found: " + foundFile.uri + " Expected: " + expected[counter].uri)
		}
		counter++
	}

	if counter != 2 {
		tests.Fatal("not all expected files were found")
	}
	cancelFunction()
}

func TestFindFiles(tests *testing.T) {

	testfile1, err := CreateTestFile("TestFindFiles/testfile1.abc")
	if err != nil {
		tests.Fatal(err.Error())
	}
	defer os.RemoveAll("TestFindFiles")

	target := Target{
		Location: filepath.Dir(testfile1),
		Match:    ".*.abc$",
	}

	expected := []foundFile{{target: &target, uri: testfile1}}

	foundFiles := make(chan foundFile, 10)
	client := linkClient{}
	currentContext, cancelFunction := context.WithCancel(context.Background())
	client.currentContext = currentContext
	client.statusRecorder = createStatusRecorder(currentContext)
	err = client.findFiles(target, &foundFiles)
	if err != nil && err != io.EOF {
		tests.Fatal("Unexpected err return from findfiles: " + err.Error())
	}
	close(foundFiles)

	counter := 0
	for foundFile := range foundFiles {

		if foundFile.uri != expected[counter].uri {
			tests.Fatal("Found: " + foundFile.uri + " Expected: " + expected[counter].uri)
		}
		counter++
	}

	if counter < 1 {
		tests.Fatal("Found: " + strconv.Itoa(counter) + " Expected: " + strconv.Itoa(len(expected)))
	}
	cancelFunction()
}

func TestGetMetadatFromFoundFile(tests *testing.T) {
	foundFile := foundFile{}
	metadata := foundFile.getMetadata()
	if len(metadata) != 0 {
		tests.Fatal("nil founds created meta data")
	}

	foundFile.uri = "//share1/dir34/file12.abc"
	date, _ := time.Parse("YYYYMMdd-HHmmss", "20170511-103614")
	foundFile.modifyTime = date
	foundFile.size = 2000
	foundFile.target = &Target{
		Active:     true,
		DataFormat: "abc",
		DataType:   "test",
		Name:       "testTarget1",
		Tenant:     "testCustomer",
		PathEncodedMetaDataTags: []PathEncodedMetaDataTag{
			{
				Tag:   "file_number",
				Match: `.*[^\d](\d+)\.abc`},
			{
				Tag:   "dir_number",
				Match: `\/.*[^\d](\d*)\/[^\/]*$`},
		},
	}

	metadata = foundFile.getMetadata()
	if metadata[trueconnect.TenantID].Value != "testCustomer" {
		tests.Fatal("Tennant ID metadata not populated")
	}
	if metadata[trueconnect.DataType].Value != "test" {
		tests.Fatal("DataType metadata not populated")
	}
	if metadata[trueconnect.FileFormat].Value != "abc" {
		tests.Fatal("FileFormat metadata not populated")
	}
	if metadata[trueconnect.OriginalFileName].Value != "//share1/dir34/file12.abc" {
		tests.Fatal("Filename metadata not populated")
	}
	host, _ := os.Hostname()
	if metadata[sourceHost].Value != host {
		tests.Fatal("Hostname metadata not populated")
	}
	if metadata[fileSize].Value != "2000" {
		tests.Fatal("file size metadata not populated")
	}
	if metadata[lastModifiedDate].Value != date.Format(time.RFC3339) {
		tests.Fatal("Last modified metadata not populated")
	}
	if metadata["file_number"].Value != "12" {
		tests.Fatal((metadata)["file_number"].Value + ": Path encoded meta data file_number not populated")
	}
	if metadata["dir_number"].Value != "34" {
		tests.Fatal((metadata)["dir_number"].Value + ": Path encoded meta data dir_number not populated")
	}
}

func TestStaticMetaDataOnFoundFile(tests *testing.T) {
	foundFile := foundFile{}

	foundFile.uri = "//share1/dir34/file12.abc"
	date, _ := time.Parse("YYYYMMdd-HHmmss", "20170511-103614")
	foundFile.modifyTime = date
	foundFile.size = 2000
	foundFile.target = &Target{
		Active:     true,
		DataFormat: "abc",
		DataType:   "test",
		Name:       "testTarget1",
		Tenant:     "testCustomer",
		StaticTags: []StaticMetaData{
			{Tag: "testTag", Value: "hello"},
		},
	}
	metadata := foundFile.getMetadata()
	if metadata["testTag"].Value != "hello" {
		tests.Fatal("static tag not found")
	}
}

func CreateTestFile(createFileName string) (string, error) {
	fileName, err := filepath.Abs(createFileName)
	testdir := filepath.Dir(fileName)
	if err == nil {
		err = os.MkdirAll(testdir, 0775)
		if err == nil {
			file, err := os.Create(fileName)
			if err == nil {
				err = file.Close()
			}
		}
	}
	if err != nil {
		return "", err
	}

	return fileName, nil
}
