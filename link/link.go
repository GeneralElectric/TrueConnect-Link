// Package link is the client core, it initialises all the targets to be search from the config and sets ups all the workers
package link

import (
	"context"
	"encoding/json"
	"github.build.ge.com/ADF/trueconnect-link/trueconnect"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Additional Metadata provided by this client
const (
	sourceHost                   = "source_host"
	fileSize                     = "file_size"
	lastModifiedDate             = "last_modified_date"
	sha256Hash                   = "sha_256"
	uploadSuccess                = "Success"
	fileUploadOpp                = "FileUpload"
	systemName                   = "TrueConnect-Link"
	mainOperation                = "Main"
	startingStatus               = "Starting"
	startedStatus                = "Started"
	skippedStatus                = "Skipped"
	operationLoadConfig          = "LoadConfig"
	failedStatus                 = "Failed"
	oppTrueConnectAuthentication = "trueConnectAuthentication"
	searchingStatus              = "Searching"
	statSearchComplete           = "SearchComplete"
	statStopping                 = "Stopping"
	partialStatus                = "Partial"
	commandOperation             = "CommandOnUpload"
)

type linkClient struct {
	configuration        Configuration
	isStopping           bool
	currentContext       context.Context
	exitCode             int
	fileTransferRecorder fileTransferRecorder
	statusRecorder       *statusRecorder
	recorderCancel       context.CancelFunc
}

// ClientInterface is an interface that defines the publicly accessible methods of the true connect client
type ClientInterface interface {
	// Start method will search the configured targets for files extracting the appropriate metadata before uploading
	// them to TrueConnect
	Start() string
	// GetExitCode gets the exit code to report out to the operating system on completion of execution 0 = ok
	GetExitCode() int

	//used to do the final clean up after everything is done
	Dispose()
}

// NewClient function initialises a new client based on the argument given and the current context
func NewClient(ctx context.Context, args []string) (ClientInterface, error) {
	return newClientStruct(ctx, args)
}

// same as above just testable
func newClientStruct(ctx context.Context, args []string) (*linkClient, error) {
	client := &linkClient{}
	client.currentContext = ctx
	client.fileTransferRecorder = createFileTransferRecorder()
	client.configuration.getConfigurationFromArgs(args)
	fileName := client.configuration.ClientID + ".recordStatus"
	err := client.fileTransferRecorder.buildFromStatusEntry(fileName)
	if err != nil {
		return nil, err
	}
	var cancelableContext context.Context

	cancelableContext, client.recorderCancel = context.WithCancel(context.Background())
	client.statusRecorder, err = createFileStatusRecorder(cancelableContext, fileName)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (linkClient *linkClient) Dispose() {
	linkClient.recorderCancel()
}

// Start method will search the configured targets for files extracting the appropriate metadata before uploading them to
// TrueConnect
func (linkClient *linkClient) Start() string {
	linkClient.isStopping = false
	contextID := linkClient.statusRecorder.recordStatus(systemName, mainOperation, startingStatus, "", "")
	switch linkClient.configuration.command {
	case UploadCommand:
		// do nothing
		break
	case TestCommand:
		linkClient.isStopping = true
		return selfTest()
	case HelpCommand:
		return Usage
	case StartCommand:
		err := linkClient.loadConfigWithTargets()
		if err != nil {
			linkClient.statusRecorder.recordStatus(systemName, operationLoadConfig, failedStatus, contextID, err.Error())
			linkClient.isStopping = true
			linkClient.exitCode = 1
			return err.Error()
		}
		linkClient.configuration.RunAsService = false
		break
	case AutoCommand:
		err := linkClient.loadConfigWithTargets()
		if err != nil {
			linkClient.isStopping = true
			linkClient.exitCode = 1
			linkClient.statusRecorder.recordStatus(systemName, operationLoadConfig, failedStatus, contextID, err.Error())
			return err.Error()
		}
		linkClient.configuration.RunAsService = true
		break
	default:
		linkClient.statusRecorder.recordStatus(systemName, startingStatus, failedStatus, contextID, "Unrecognised Command")
		linkClient.isStopping = true
		linkClient.exitCode = 1
		return Usage
	}

	err := linkClient.authenticate()
	if err != nil {
		linkClient.statusRecorder.recordStatus(systemName, oppTrueConnectAuthentication, failedStatus, contextID, err.Error())
		linkClient.isStopping = true
		linkClient.exitCode = 1
		return err.Error()
	}

	foundFiles := linkClient.processTargets(linkClient.configuration.Targets)
	linkClient.doWork(foundFiles)
	linkClient.statusRecorder.recordStatus(systemName, mainOperation, statStopping, contextID, "")
	linkClient.isStopping = true
	return ""
}

// GetExitCode gets the exit code to report out to the operating system on completion of execution 0 = ok
func (linkClient *linkClient) GetExitCode() int {
	return linkClient.exitCode
}

func (linkClient *linkClient) doWork(foundFiles *chan foundFile) {
	if linkClient.configuration.ConcurrentUploads < 1 {
		linkClient.configuration.ConcurrentUploads = 1
	}
	var waitGroup sync.WaitGroup

	for i := 0; i < linkClient.configuration.ConcurrentUploads; i++ {
		waitGroup.Add(1)
		go func(foundFiles *chan foundFile) {

			for {
				select {
				case <-linkClient.currentContext.Done():
					waitGroup.Done()
					return
				case foundFile, isOk := <-*foundFiles:
					if !isOk {
						waitGroup.Done()
						return
					}
					uid := foundFile.hash + "~" + foundFile.uri
					foundFile.progress, isOk = linkClient.fileTransferRecorder.startRecord(uid, foundFile.progress)
					if isOk {
						linkClient.statusRecorder.recordStatus(systemName, fileUploadOpp, startedStatus, uid, foundFile.uri)
						progress, err := linkClient.upload(foundFile)
						foundFile.progress = progress
						partial := linkClient.fileTransferRecorder.stopRecord(uid, foundFile.progress)
						if err == nil {
							linkClient.statusRecorder.recordStatus(systemName, fileUploadOpp, uploadSuccess, uid, foundFile.progress.Reference)
							if foundFile.target.OnSuccess != "" {
								err := linkClient.ExecuteOnSuccess(foundFile)
								if err != nil {
									linkClient.statusRecorder.recordStatus(systemName, commandOperation, failedStatus, uid, err.Error())
								}
							}
						} else {
							if linkClient.exitCode == 0 {
								linkClient.exitCode = 2
							}
							if !partial || os.IsNotExist(err) {
								linkClient.statusRecorder.recordStatus(systemName, fileUploadOpp, failedStatus, uid, err.Error())
							} else {
								progBytes, _ := json.Marshal(progress)
								linkClient.statusRecorder.recordStatus(systemName, fileUploadOpp, partialStatus, uid, string(progBytes))
								if !linkClient.isStopping {
									linkClient.retryIn(foundFile, 120, foundFiles)
								}
							}
						}
					} else {
						linkClient.statusRecorder.recordStatus(systemName, fileUploadOpp, skippedStatus, uid, foundFile.uri)
					}
				}
			}
		}(foundFiles)
	}
	waitGroup.Wait()
}

func (linkClient *linkClient) retryIn(file foundFile, seconds int, foundFiles *chan foundFile) {
	go func() {
		select {
		case <-linkClient.currentContext.Done():
			return
		case <-time.After(time.Duration(seconds) * time.Second):
		}
		if !linkClient.isStopping {
			select {
			case <-linkClient.currentContext.Done():
				return
			case *foundFiles <- file:
			}
		}
	}()
}

func selfTest() string {
	return checkConnection() + checkAllConfigs()
}

func (linkClient *linkClient) processTargets(targets []Target) *chan foundFile {

	ff := make(chan foundFile)
	foundFiles := &ff
	go func() {
		var waitGroup sync.WaitGroup
		for _, target := range targets {
			// for closure https://stackoverflow.com/questions/26692844/captured-closure-for-loop-variable-in-go
			currentTarget := target
			// run until either (all files founds are files are found and !linkClient.configuration.RunAsService ) or linkClient.isStopping
			waitGroup.Add(1)
			// we spin of a new thread for each target so small files don't have to wait for all the large files to upload before they start
			go func() {
				defer waitGroup.Done()
				for {
					contextID := linkClient.statusRecorder.recordStatus(systemName, currentTarget.Name, searchingStatus, "", currentTarget.Location)
					err := linkClient.findFiles(currentTarget, foundFiles)
					if err != nil && err != io.EOF {
						linkClient.exitCode = 1
						if err.Error() != errTerminating {
							linkClient.statusRecorder.recordStatus(systemName, currentTarget.Name, failedStatus, contextID, err.Error())
						}
						return
					}
					linkClient.statusRecorder.recordStatus(systemName, currentTarget.Name, statSearchComplete, contextID, currentTarget.Location)
					if !linkClient.configuration.RunAsService || linkClient.isStopping {
						return
					}

					select {
					case <-linkClient.currentContext.Done():
						return
					case <-time.After(time.Duration(currentTarget.PollInterval) * time.Second):
					}
				}
			}()
		}
		waitGroup.Wait()
		close(*foundFiles)
	}()

	return foundFiles
}

func (linkClient *linkClient) authenticate() error {

	_, err := getScopes(
		linkClient.configuration.TokenURL,
		linkClient.configuration.ClientID,
		linkClient.configuration.Secret)
	if err != nil {
		return err
	}

	return nil
}

func (linkClient *linkClient) upload(foundFile foundFile) (trueconnect.UploadProgress, error) {

	meta := foundFile.getMetadata()

	tcwrapper := trueconnect.CreateWrapper(linkClient.configuration.TokenURL, linkClient.configuration.ClientID, linkClient.configuration.Secret, linkClient.configuration.Endpoint, 8000000)

	progress, err := tcwrapper.PostToTC(linkClient.currentContext, foundFile.progress, foundFile.uri, meta)
	if err != nil {
		progress.FailedAttempts++
	}
	return progress, err
}

func (linkClient *linkClient) ExecuteOnSuccess(foundFile foundFile) error {

	cmd := exec.Command(foundFile.target.OnSuccess, foundFile.uri, foundFile.progress.Reference)
	return cmd.Run()
}
