package trueconnect

import (
	"bufio"
	"context"
	"crypto/md5" // #nosec
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"testing"
)

const (
	errTest   = "errTest"
	ENVsecret = "secret"
)

func TestCopyContextStop(tests *testing.T) {
	var err error
	padding := make([]byte, copyBufferSize)
	rand.Read(padding)
	ctx, cancel := context.WithCancel(context.Background())
	sourceReader, sourceWriter := io.Pipe()
	destReader, destWriter := io.Pipe()
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		sourceWriter.Write(padding)
		sourceWriter.Write(padding)
		sourceWriter.Write(padding)
		cancel()
		sourceWriter.Write(padding)
	}()
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(destReader)
		for scanner.Scan() {
			scanner.Text()
		}
	}()
	_, err = copyBufferWithCTX(ctx, destWriter, sourceReader, 8000000)

	sourceWriter.Close()
	destWriter.Close()
	if err.Error() != errStreamInterrupted {
		tests.Fatal("stream not interrupted")
	}
	wg.Wait()
}

func TestCopyEOFStop(tests *testing.T) {
	var err error
	padding := make([]byte, copyBufferSize)
	rand.Read(padding)
	sourceReader, sourceWriter := io.Pipe()
	destReader, destWriter := io.Pipe()
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for err == nil {
			_, err = copyBufferWithCTX(context.Background(), destWriter, sourceReader, 8000000)
		}
		destWriter.Close()
		wg.Done()
	}()
	go func() {
		scanner := bufio.NewScanner(destReader)
		for scanner.Scan() {
			scanner.Text()
		}
		wg.Done()
	}()
	sourceWriter.Write(padding)
	sourceWriter.Write([]byte("\ntesting1\n"))
	sourceWriter.Write(padding)
	sourceWriter.Write([]byte("\ntesting2\n"))
	sourceWriter.Write(padding)
	sourceWriter.Write([]byte("\ntesting3\n"))
	sourceWriter.Close()
	wg.Wait()
	if err != io.EOF {
		tests.Fatal("stream did not reach end")
	}
}

func TestCopyErrorPassThrough(tests *testing.T) {
	var err error
	padding := make([]byte, copyBufferSize)
	rand.Read(padding)
	sourceReader, sourceWriter := io.Pipe()
	destReader, destWriter := io.Pipe()
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for err == nil {
			_, err = copyBufferWithCTX(context.Background(), destWriter, sourceReader, 8000000)
		}
		destWriter.Close()
		wg.Done()
	}()
	go func() {
		scanner := bufio.NewScanner(destReader)
		for scanner.Scan() {
			scanner.Text()
		}
		wg.Done()
	}()
	sourceWriter.Write(padding)
	sourceWriter.Write([]byte("\ntesting1\n"))
	sourceWriter.Write(padding)
	sourceWriter.Write([]byte("\ntesting2\n"))
	sourceWriter.Write(padding)
	sourceWriter.Write([]byte("\ntesting3\n"))
	sourceWriter.CloseWithError(fmt.Errorf(errTest))
	wg.Wait()
	if err.Error() != errTest {
		tests.Fatal("error not passed through")
	}
}

func TestCopyHash(tests *testing.T) {
	var err error
	var hash string
	padding := make([]byte, copyBufferSize-1000)
	rand.Read(padding)
	md5er := md5.New()
	md5er.Write(padding)
	prehash := hex.EncodeToString(md5er.Sum(nil))
	sourceReader, sourceWriter := io.Pipe()
	destReader, destWriter := io.Pipe()
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for err == nil {
			hash, err = copyBufferWithCTX(context.Background(), destWriter, sourceReader, 8000000)
		}
		destWriter.Close()
		wg.Done()
	}()
	go func() {
		scanner := bufio.NewScanner(destReader)
		for scanner.Scan() {
			scanner.Text()
		}
		wg.Done()
	}()
	sourceWriter.Write(padding)
	sourceWriter.Close()
	wg.Wait()
	if hash != prehash {
		tests.Fatal("incorrect hash")
	}
}

func TestStartPartsFORREAL(tests *testing.T) {

	if os.Getenv(ENVsecret) != "" {
		wrap := Wrapper{
			ChunkSize: 6000000,
			Endpoint:  "https://trueconnect-dev.run.aws-usw02-pr.ice.predix.io",
			Secret:    os.Getenv(ENVsecret),
			ClientID:  "aviation_trueconnect-tenancytest3_dev",
			TokenURL:  "https://a8a2ffc4-b04e-4ec1-bfed-7a51dd408725.predix-uaa.run.aws-usw02-pr.ice.predix.io/oauth/token",
		}

		meta := make(map[string]MetadataValue)
		meta[OriginalFileName] = MetadataValue{Value: "43-Texture_of_the_landscape.tif", Immutable: true, Notify: false}
		meta[TenantID] = MetadataValue{Value: "testguid3", Immutable: true, Notify: false}
		meta[DataType] = MetadataValue{Value: "testdata", Immutable: true, Notify: false}
		meta[FileFormat] = MetadataValue{Value: "junk", Immutable: true, Notify: false}
		str, err := wrap.startParts(context.Background(), meta)
		if err != nil {
			tests.Fatal(err)
		}

		progress := UploadProgress{Reference: str}

		progress, err = wrap.uploadParts(context.Background(), "C:\\Users\\NGGM3GN\\Pictures\\43-Texture_of_the_landscape.tif", progress)

		if err != nil {
			tests.Fatal(err)
		}

		meta = make(map[string]MetadataValue)
		meta["hatsize"] = MetadataValue{Value: `7 3/4`, Immutable: true, Notify: true}
		progress, err = wrap.completeUpload(context.Background(), progress, meta)

		if err != nil {
			tests.Fatal(err)
		}

		if progress.Part != 4 {
			tests.Fatal(fmt.Sprintf("Parts = %d",progress.Part))
		}
	} else {
		tests.Skip("uploads real file to TC")
	}
}
