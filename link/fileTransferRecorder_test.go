package link

import (
	"context"
	"encoding/json"
	"github.com/GeneralElectric/TrueConnect-Link/trueconnect"
	"reflect"
	"testing"
	"time"
)

func TestNotDuplicateRecord(tests *testing.T) {
	recorder := createFileTransferRecorder()
	first := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 0}
	p, isOk := recorder.startRecord("abc123", first)
	if !isOk || !reflect.DeepEqual(first, p) {
		tests.Fatal("Could Not create file transfer record")
	}

	second := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 0}
	_, isOk = recorder.startRecord("abc123", second)
	if isOk {
		tests.Fatal("Did not prevent duplicate")
	}
}

func TestCompleteNotReturnPartialRecord(tests *testing.T) {
	recorder := createFileTransferRecorder()
	first := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 0}
	p, isOk := recorder.startRecord("abc123", first)
	if !isOk || !reflect.DeepEqual(first, p) {
		tests.Fatal("Could Not create file transfer record")
	}

	second := trueconnect.UploadProgress{Complete: true, Reference: "abc", Part: 9, FailedAttempts: 0}
	isPartial := recorder.stopRecord("abc123", second)
	if isPartial {
		tests.Fatal("Stop with complete progress returned partial")
	}
}

func TestIncompleteReturnsPartialRecord(tests *testing.T) {
	recorder := createFileTransferRecorder()
	first := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 0}
	p, isOk := recorder.startRecord("abc123", first)
	if !isOk || !reflect.DeepEqual(first, p) {
		tests.Fatal("Could Not create file transfer record")
	}

	second := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 0}
	isPartial := recorder.stopRecord("abc123", second)
	if !isPartial {
		tests.Fatal("Stop with incomplete progress returned non-partial")
	}
}

func TestNotRedoCompletedRecord(tests *testing.T) {
	recorder := createFileTransferRecorder()
	first := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 0}
	p, isOk := recorder.startRecord("abc123", first)
	if !isOk || !reflect.DeepEqual(first, p) {
		tests.Fatal("Could Not create file transfer record")
	}

	second := trueconnect.UploadProgress{Complete: true, Reference: "abc", Part: 9, FailedAttempts: 0}
	isPartial := recorder.stopRecord("abc123", second)
	if isPartial {
		tests.Fatal("Stop with complete progress returned partial")
	}

	p, isOk = recorder.startRecord("abc123", first)
	if isOk {
		tests.Fatal("allowed start on completeed upload")
	}
}

func TestResumeIncompletedRecord(tests *testing.T) {
	recorder := createFileTransferRecorder()
	first := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 0}
	p, isOk := recorder.startRecord("abc123", first)
	if !isOk || !reflect.DeepEqual(first, p) {
		tests.Fatal("Could Not create file transfer record")
	}

	second := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 0}
	isPartial := recorder.stopRecord("abc123", second)
	if !isPartial {
		tests.Fatal("Stop with incomplete progress returned non-partial")
	}

	p, isOk = recorder.startRecord("abc123", first)
	if !isOk {
		tests.Fatal("allowed start on completeed upload")
	}

	if !reflect.DeepEqual(p, second) {
		tests.Fatal("start did not return previous progress")
	}
}

func TestCompleteOverridesRetriesRecord(tests *testing.T) {
	recorder := createFileTransferRecorder()
	first := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 0}
	p, isOk := recorder.startRecord("abc123", first)
	if !isOk || !reflect.DeepEqual(first, p) {
		tests.Fatal("Could Not create file transfer record")
	}

	second := trueconnect.UploadProgress{Complete: true, Reference: "abc", Part: 9, FailedAttempts: 8}
	isPartial := recorder.stopRecord("abc123", second)
	if isPartial {
		tests.Fatal("Stop with complete progress returned partial")
	}

	p, isOk = recorder.startRecord("abc123", first)
	if isOk {
		tests.Fatal("allowed start on completeed upload")
	}
}

func TestExceededRetriesTriggersStartOver(tests *testing.T) {
	recorder := createFileTransferRecorder()
	first := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 0}
	p, isOk := recorder.startRecord("abc123", first)
	if !isOk || !reflect.DeepEqual(first, p) {
		tests.Fatal("Could Not create file transfer record")
	}

	second := trueconnect.UploadProgress{Complete: false, Reference: "abc", Part: 9, FailedAttempts: 3}
	isPartial := recorder.stopRecord("abc123", second)
	if isPartial {
		tests.Fatal("Stop too many retries returned partial")
	}

	p, isOk = recorder.startRecord("abc123", first)
	if !isOk && reflect.DeepEqual(p, first) {
		tests.Fatal("Didn't start over on over tried")
	}
}

func TestBuildFromLog(tests *testing.T) {
	currentContext, cancelFunction := context.WithCancel(context.Background())
	client := linkClient{}
	client.currentContext = currentContext
	var err error
	client.statusRecorder, err = createFileStatusRecorder(currentContext, "test.recordStatus")
	if err != nil {
		tests.Fatal(err)
	}
	//defer os.Remove("test.recordStatus")
	uid1 := "deadface123" + "~" + "C/:test/test.file"
	uid2 := "deadface123" + "~" + "C/:test/test.xxx"
	uid3 := "ace99~c/:something.one"
	progress := trueconnect.UploadProgress{Reference: "xyz", Part: 2}
	progBytes, _ := json.Marshal(progress)
	client.statusRecorder.recordStatus(systemName, fileUploadOpp, partialStatus, uid3, string(progBytes))
	client.statusRecorder.recordStatus(systemName, fileUploadOpp, uploadSuccess, uid1, "testref")
	cancelFunction()
	time.Sleep(time.Second * 1)

	client.fileTransferRecorder = createFileTransferRecorder()
	err = client.fileTransferRecorder.buildFromStatusEntry("test.recordStatus")
	if err != nil {
		tests.Fatal(err.Error())
	}
	if _, isOk := client.fileTransferRecorder.startRecord(uid1, trueconnect.UploadProgress{}); isOk {
		tests.Fatal("Failed to prevent duplicate record")
	}

	if _, isOk := client.fileTransferRecorder.startRecord(uid2, trueconnect.UploadProgress{}); !isOk {
		tests.Fatal("Failed to allow record")
	}

	if p, isOk := client.fileTransferRecorder.startRecord(uid3, trueconnect.UploadProgress{}); !isOk && reflect.DeepEqual(p, progress) {
		tests.Fatal("Failed to resume")
	}
}
