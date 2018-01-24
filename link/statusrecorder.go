package link

import (
	"context"
	"encoding/csv"
	"github.com/google/uuid"
	"io"
	"log"
	"os"
	"time"
)

type statusRecorder struct {
	currentContext context.Context
	statusChannel  chan StatusRecordEntry
	fileToClose    *os.File
}

// StatusRecordEntry is used to record the change in state in the application and is used to record such things as the
// completion of a file upload
type StatusRecordEntry struct {
	// The time at which the state changed
	Time time.Time

	// The system whose state changed
	System string

	// The operation or action that caused the change
	Operation string

	// The effect of the change
	Status string

	// Individual item or instance of operation that was changed
	ContextID string

	// extra information relating to the change
	Comments string
}

// StatusRecordEntryFromLine will take the parts of a line of text produced by a CSV reader that parses status entries from TrueConnect-Link and turn them
// into a StatusRecordEntry structure
func StatusRecordEntryFromLine(parts []string) StatusRecordEntry {
	var statusRecordEntry StatusRecordEntry
	if parts == nil || len(parts) < 6 {
		return statusRecordEntry
	}
	t, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return statusRecordEntry
	}
	statusRecordEntry.Time = t
	statusRecordEntry.System = parts[1]
	statusRecordEntry.Operation = parts[2]
	statusRecordEntry.Status = parts[3]
	statusRecordEntry.ContextID = parts[4]
	statusRecordEntry.Comments = parts[5]
	return statusRecordEntry
}

// StatusRecordToLine will take a StatusRecordEntry and turn it to a CSV line
func (recordEntry *StatusRecordEntry) StatusRecordToLine() []string {
	return []string{
		recordEntry.Time.Format(time.RFC3339),
		recordEntry.System,
		recordEntry.Operation,
		recordEntry.Status,
		recordEntry.ContextID,
		recordEntry.Comments,
	}
}

func createStatusRecorder(ctx context.Context) *statusRecorder {
	return createStatusWriter(ctx, os.Stdout)
}

func createFileStatusRecorder(ctx context.Context, fileName string) (*statusRecorder, error) {
	statusFile, err := os.OpenFile(fileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	thisRecorder := createStatusWriter(ctx, statusFile)
	thisRecorder.fileToClose = statusFile
	return thisRecorder, nil
}

func createStatusWriter(ctx context.Context, writer io.Writer) *statusRecorder {
	thisRecorder := statusRecorder{}
	thisRecorder.currentContext = ctx
	thisRecorder.statusChannel = make(chan StatusRecordEntry)
	go func(currentStatusChannel chan StatusRecordEntry) {
		csvWriter := csv.NewWriter(writer)
		defer close(currentStatusChannel)
		defer csvWriter.Flush()

		for {
			select {
			case statusEntry, isOk := <-currentStatusChannel:
				if !isOk {
					// if the channel has closed
					csvWriter.Flush()
					return
				}
				err := csvWriter.Write(statusEntry.StatusRecordToLine())
				if err != nil {
					err := csvWriter.Write(statusEntry.StatusRecordToLine())
					if err != nil {
						log.Fatal(err)
					}
				}
				csvWriter.Flush()
				break
			case <-thisRecorder.currentContext.Done():
				csvWriter.Flush()
				if thisRecorder.fileToClose != nil {
					err := thisRecorder.fileToClose.Close()
					if err != nil {
						log.Fatal(err)
					}
				}
				return
			}
		}
	}(thisRecorder.statusChannel)

	return &thisRecorder
}

func (statusRecorder *statusRecorder) recordStatus(system string, operation string, status string, contextID string, comments string) string {
	statusEntry := StatusRecordEntry{
		Time:      time.Now().UTC(),
		System:    system,
		Operation: operation,
		Status:    status,
		ContextID: contextID,
		Comments:  comments,
	}

	if statusEntry.ContextID == "" {
		statusEntry.ContextID = uuid.New().String()
	}

	select {

	case statusRecorder.statusChannel <- statusEntry:
		break
	case <-statusRecorder.currentContext.Done():
		break
	}
	return statusEntry.ContextID
}
