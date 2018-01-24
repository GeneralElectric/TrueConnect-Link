package link

import (
	"context"
	"encoding/csv"
	"os"
	"testing"
	"time"
)

func TestCreateLogger(tests *testing.T) {

	currentContext, cancelFunction := context.WithCancel(context.Background())
	client := linkClient{}
	client.currentContext = currentContext
	var err error
	client.statusRecorder, err = createFileStatusRecorder(currentContext, "test.csv")
	if err != nil {
		tests.Fatal(err)
	}
	client.statusRecorder.recordStatus("Test1", "testing", "tested", "", "it worked")
	cancelFunction()
	time.Sleep(time.Second * 1)
	file, err := os.Open("test.csv")
	if err != nil {
		tests.Fatal(err.Error())
	}
	defer file.Close()
	var lastRecord StatusRecordEntry
	var line []string
	reader := csv.NewReader(file)
	for {
		line, err = reader.Read()
		if err != nil {
			break
		}
		lastRecord = StatusRecordEntryFromLine(line)
	}

	duration := time.Since(lastRecord.Time)

	if duration > time.Duration(time.Second*60) {
		tests.Fatal("Cannot find last recorded entry")
	}

}
