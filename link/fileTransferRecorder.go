package link

import (
	"encoding/csv"
	"encoding/json"
	"github.build.ge.com/ADF/trueconnect-link/trueconnect"
	"io"
	"os"
	"sync"
)

type fileTransferRecorder struct {
	records                 map[string]liveUploadProgress
	fileTransferRecordMutex *sync.Mutex
}

type liveUploadProgress struct {
	progress   *trueconnect.UploadProgress
	inProgress bool
}

func createFileTransferRecorder() fileTransferRecorder {
	recorder := fileTransferRecorder{}
	recorder.fileTransferRecordMutex = &sync.Mutex{}
	recorder.records = make(map[string]liveUploadProgress)
	return recorder
}

func (recorder *fileTransferRecorder) buildFromStatusEntry(statusFileName string) error {
	records := make(map[string]liveUploadProgress)
	statusFile, err := os.Open(statusFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	defer statusFile.Close()
	csvReader := csv.NewReader(statusFile)
	for {
		var line []string
		line, err = csvReader.Read()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		statusEntry := StatusRecordEntryFromLine(line)
		if statusEntry.Operation == fileUploadOpp {
			if statusEntry.Status == uploadSuccess {
				records[statusEntry.ContextID] = liveUploadProgress{progress: &trueconnect.UploadProgress{Complete: true}}
			}
			if statusEntry.Status == partialStatus {
				var progress trueconnect.UploadProgress
				err := json.Unmarshal([]byte(statusEntry.Comments), &progress)
				if err != nil {
					return nil
				}
				records[statusEntry.ContextID] = liveUploadProgress{progress: &progress}
			}
		}
	}
	recorder.records = records
	return nil
}

func (recorder *fileTransferRecorder) startRecord(record string, progress trueconnect.UploadProgress) (trueconnect.UploadProgress, bool) {
	isOk := false
	recorder.fileTransferRecordMutex.Lock()
	{
		currentProg, exists := recorder.records[record]
		if !exists {
			recorder.records[record] = liveUploadProgress{progress: &progress, inProgress: !progress.Complete}
			isOk = true
		} else if !currentProg.inProgress && !currentProg.progress.Complete {
			isOk = true
			a := recorder.records[record]
			a.inProgress = true
			recorder.records[record] = a
		}
	}
	recorder.fileTransferRecordMutex.Unlock()
	return *recorder.records[record].progress, isOk
}

func (recorder *fileTransferRecorder) stopRecord(record string, progress trueconnect.UploadProgress) bool {
	isIncommplete := false
	recorder.fileTransferRecordMutex.Lock()
	{
		currentProg, exists := recorder.records[record]
		if exists && currentProg.inProgress {
			if progress.Reference == "" {
				if currentProg.progress.Reference == "" {
					delete(recorder.records, record)
				}
			} else {
				if progress.Complete {
					recorder.records[record] = liveUploadProgress{progress: &progress, inProgress: false}
				} else if progress.FailedAttempts > 2 {
					delete(recorder.records, record)
				} else {
					isIncommplete = true
					recorder.records[record] = liveUploadProgress{progress: &progress, inProgress: false}
				}
			}
		}
	}
	recorder.fileTransferRecordMutex.Unlock()
	return isIncommplete
}
