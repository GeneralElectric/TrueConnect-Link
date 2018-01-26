package link

import (
	"context"
	"github.com/GeneralElectric/TrueConnect-Link/trueconnect"
	"os"
	"runtime"
	"testing"
	"time"
)

func init() {
	trueconnect.CreateWrapper = func(TokenURL string, ClientID string, Secret string, Endpoint string, ChunkSize int64) trueconnect.WrapperInterface {
		return &proxyTc{behaviour: TokenURL}
	}
}

type proxyTc struct {
	behaviour string
}

func (wrapper *proxyTc) PostToTC(ctx context.Context, progress trueconnect.UploadProgress, filenamePath string, meta map[string]trueconnect.MetadataValue) (trueconnect.UploadProgress, error) {
	progress.Reference = "proxygenerated"
	if wrapper.behaviour == "win" {
		progress.Complete = true
	}
	return progress, nil
}

func TestNewClient(tests *testing.T) {
	args := []string{
		"-u:User1",
		"-c:Upload",
		"-p:p4ssw0rd",
		"-tenant:tenant1",
		"-datatype:datatype1",
		"-dataformat:dataformat1",
		"-path:boiled.egg",
		"-e:http://www.somesite.com/tc?action=upload",
		"-tokurl:https://some.real.long.url.that.looks.meaningless/stuff",
	}
	currentContext, cancelFunction := context.WithCancel(context.Background())
	client, err := newClientStruct(currentContext, args)
	cancelFunction()
	client.Dispose()
	time.Sleep(time.Second * 2)
	if err != nil {
		tests.Fatal(err)
	}

	err = os.Remove(client.statusRecorder.fileToClose.Name())
	if err != nil {
		tests.Fatal(err)
	}

	if client.configuration.ClientID != "User1" {
		tests.Fatal("username not set")
	}
}

func TestExecuteOnSucess(tests *testing.T) {
	if runtime.GOOS == "windows" {
		client := linkClient{}
		file, err := os.Create("callAfter.cmd")
		if err != nil {
			tests.Fatal(err)
		}
		defer os.Remove("callAfter.cmd")

		file.WriteString("copy /y nul %1.%2")
		file.Close()

		err = client.ExecuteOnSuccess(foundFile{progress: trueconnect.UploadProgress{Complete: true, Reference: "worked"}, uri: "hello", target: &Target{OnSuccess: "callAfter.cmd"}})
		if err != nil {
			tests.Fatal(err)
		}
		time.Sleep(time.Second)
		if _, err := os.Stat("hello.worked"); os.IsNotExist(err) {
			tests.Fatal("Could not find file hello.worked")
		}

		os.Remove("hello.worked")
	} else {
		tests.Skip("windows test")
	}
}
