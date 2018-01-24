package trueconnect

import (
	"bytes"
	"context"
	"crypto/md5" // #nosec
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2/clientcredentials"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"sync"
)

const (
	copyBufferSize       = int64(32 * 1024)
	errStreamInterrupted = "StreamInterrupted"
	urlUploadBase        = `/api/v1/files`
)

type WrapperInterface interface {
	PostToTC(ctx context.Context, progress UploadProgress, filenamePath string, meta map[string]MetadataValue) (UploadProgress, error)
}

var CreateWrapper = initWrapperfunc

func initWrapperfunc(TokenURL string, ClientID string, Secret string, Endpoint string, ChunkSize int64) WrapperInterface {
	return &Wrapper{TokenURL: TokenURL, ClientID: ClientID, Secret: Secret, Endpoint: Endpoint, ChunkSize: ChunkSize}
}

// UploadProgress is a struct used to store the state of a file upload
type UploadProgress struct {
	// This is the file reference used by TrueConnect for a partial or complete file upload
	Reference string `json:"reference"`
	// The index of the last sucessfully uploaded part of the file
	Part int `json:"part"`
	// Set true when upload has completed
	Complete bool `json:"complete"`
	// The number of times a part has failed to upload since the last successful uploaded part
	FailedAttempts int `jason:"fails"`
}

// Wrapper is struct stores state associate with a connection to a TrueConnect instance and an client
type Wrapper struct {
	// URL from which the UAA token is sort
	TokenURL string
	// Name used to identify the client
	ClientID string
	// Secret used to authenticate the client with the UAA service
	Secret string
	// The url of the true connect service
	Endpoint string
	// The size of the individual parts of a multipart upload
	ChunkSize int64
}

// PostToTC post a file to the TrueConnect service
// ctx is the context used to govern the timeout and cancellation functionality of the post operation
func (wrapper *Wrapper) PostToTC(ctx context.Context, progress UploadProgress, filenamePath string, meta map[string]MetadataValue) (UploadProgress, error) {
	var err error
	if progress.Complete {
		return progress, nil
	}

	fileInf, err := os.Stat(filenamePath)
	if err != nil {
		return progress, err
	}

	size := fileInf.Size()
	if size <= wrapper.ChunkSize { // its small
		return wrapper.uploadInOne(ctx, filenamePath, meta, size)
	}

	notifiables := make(map[string]MetadataValue)
	for key, value := range meta {
		if value.Notify {
			notifiables[key] = value
		}
	}
	for key := range notifiables {
		delete(meta, key)
	}

	if progress.Reference == "" { // its new
		progress.Reference, err = wrapper.startParts(ctx, meta)
		if err != nil {
			return progress, err
		}
	}

	progress, err = wrapper.uploadParts(ctx, filenamePath, progress)
	if err != nil {
		return progress, err
	}

	return wrapper.completeUpload(ctx, progress, notifiables)
}

func (wrapper *Wrapper) startParts(ctx context.Context, meta map[string]MetadataValue) (string, error) {

	client := wrapper.getHTTPClient(ctx)

	// ReadWriter for the request body
	body := &bytes.Buffer{}

	err := json.NewEncoder(body).Encode(meta)
	if err != nil {
		return "", err
	}

	// Create a request
	// Cannot use client.Post here, as we need to set headers
	req, err := http.NewRequest("POST", wrapper.Endpoint+urlUploadBase+"/chunked", body)
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	// Make the request
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}

	// We have a response body if we get here, so make sure we close it when done
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(response.Status, response.StatusCode)
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(response.Body)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
func (wrapper *Wrapper) getHTTPClient(ctx context.Context) *http.Client {
	creds := clientcredentials.Config{
		TokenURL:     wrapper.TokenURL,
		ClientID:     wrapper.ClientID,
		ClientSecret: wrapper.Secret,
	}
	return creds.Client(ctx)
}

func (wrapper *Wrapper) uploadInOne(ctx context.Context, filenamePath string, meta map[string]MetadataValue, size int64) (progress UploadProgress, err error) {
	file, err := os.Open(filenamePath)
	if err != nil {
		return progress, err
	}
	defer file.Close()

	client := wrapper.getHTTPClient(ctx)

	pipeOut, pipeIn := io.Pipe()

	// Writer to build the request
	writer := multipart.NewWriter(pipeIn)
	done := make(chan error, 1)

	var fm FileMetadata
	go func() {
		defer close(done)
		// Create a request
		// Cannot use client.Post here, as we need to set headers
		req, err := http.NewRequest("POST", fmt.Sprintf("%s%s?size=%d", wrapper.Endpoint, urlUploadBase, size), pipeOut)
		if err != nil {
			done <- err
			return
		}
		req = req.WithContext(ctx)
		req.Header.Add("Content-Type", writer.FormDataContentType())

		// Make the request
		response, err := client.Do(req)
		if err != nil {
			done <- err
			return
		}

		// We have a response body if we get here, so make sure we close it when done
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			done <- fmt.Errorf(response.Status, response.StatusCode)
			return
		}

		// Deserialize the response
		err = json.NewDecoder(response.Body).Decode(&fm)
		if err != nil {
			done <- err
			return
		}
	}()

	// Add the metadata
	// Cannot use w.CreateFormField() because we need to set a Content-Type header
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Type", "application/json")
	mh.Set("Content-Disposition", "form-data; name=\"metadata\"")
	metaWriter, err := writer.CreatePart(mh)
	if err != nil {
		return progress, err
	}

	err = json.NewEncoder(metaWriter).Encode(meta)
	if err != nil {
		return progress, err
	}

	// Add the file
	f, err := writer.CreateFormFile("input_file", filenamePath)
	if err != nil {
		if err == io.ErrClosedPipe {
			err = <-done
		}
		return progress, err
	}

	_, err = copyBufferWithCTX(ctx, f, file, wrapper.ChunkSize+1)
	if err != nil {
		if err != io.EOF {
			err2 := pipeOut.CloseWithError(err)
			if err2 != nil {
				return progress, err2
			}
			return progress, err
		}
	}

	// Finalize the body
	err = writer.Close()
	if err != nil {
		return progress, err
	}

	err = pipeIn.Close()
	if err != nil {
		return progress, err
	}

	err = <-done
	if err != nil {
		return progress, err
	}

	progress.Reference = fm.DataStoreRef
	progress.Complete = true
	return progress, nil
}

func (wrapper *Wrapper) uploadParts(ctx context.Context, filenamePath string, progress UploadProgress) (UploadProgress, error) {
	var err error
	file, err := os.Open(filenamePath)
	if err != nil {
		return progress, err
	}
	defer file.Close()

	client := wrapper.getHTTPClient(ctx)

	fileInf, err := os.Stat(filenamePath)
	if err != nil {
		return progress, err
	}

	size := fileInf.Size()

	remains := size % wrapper.ChunkSize
	numberOfParts := size / wrapper.ChunkSize
	if remains > 0 {
		numberOfParts++
	}
	for err == nil && (int64(progress.Part) < numberOfParts) {
		_, err := file.Seek(int64(progress.Part)*wrapper.ChunkSize, 0)
		if err != nil {
			return progress, err
		}
		pipeOut, pipeIn := io.Pipe()

		// Writer to build the request
		writer := multipart.NewWriter(pipeIn)
		done := make(chan error, 1)
		partSize := wrapper.ChunkSize
		if (int64(progress.Part+1) * wrapper.ChunkSize) > size {
			partSize = remains
		}
		returnedMD5 := ""
		go func() {
			defer close(done)
			// Create a request
			url := fmt.Sprintf("%s/api/v1/files/chunked/%s/part/%d?size=%d",
				wrapper.Endpoint, progress.Reference, progress.Part+1, partSize)
			// Cannot use client.Post here, as we need to set headers
			req, err := http.NewRequest("POST", url, pipeOut)
			if err != nil {
				done <- err
				return
			}
			req = req.WithContext(ctx)
			req.Header.Add("Content-Type", writer.FormDataContentType())

			// Make the request
			response, err := client.Do(req)
			if err != nil {
				done <- err
				return
			}

			// We have a response body if we get here, so make sure we close it when done
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK {
				done <- fmt.Errorf(response.Status, response.StatusCode)
				return
			}

			data, err := ioutil.ReadAll(response.Body)
			if err != nil {
				done <- err
				return
			}

			var msgMap map[string]string

			err = json.Unmarshal(data, &msgMap)
			if err != nil {
				done <- err
				return
			}

			returnedMD5 = msgMap["md5_checksum"]
		}()

		// Add the file
		f, err := writer.CreateFormFile("input_file", filenamePath)
		if err != nil {
			if err == io.ErrClosedPipe {
				err = <-done
			}
			return progress, err
		}

		hash, err := copyBufferWithCTX(ctx, f, file, wrapper.ChunkSize)
		if err != nil {
			if err != io.EOF {
				err2 := pipeOut.CloseWithError(err)
				if err2 != nil {
					return progress, err2
				}
				return progress, err
			}
		}

		err = writer.WriteField("md5hash", hash)
		if err != nil {
			return progress, err
		}

		// Finalize the body
		err = writer.Close()
		if err != nil {
			return progress, err
		}

		err = pipeIn.Close()
		if err != nil {
			return progress, err
		}

		err = <-done
		if err != nil {
			return progress, err
		}

		if hash != returnedMD5 {
			return progress, fmt.Errorf("MD5Hash not matched uploading file part %d of %d", progress.Part, numberOfParts)
		}

		progress.FailedAttempts = 0
		progress.Part++
	}

	return progress, nil
}

func (wrapper *Wrapper) completeUpload(ctx context.Context, progress UploadProgress, meta map[string]MetadataValue) (UploadProgress, error) {
	client := wrapper.getHTTPClient(ctx)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Type", "application/json")
	mh.Set("Content-Disposition", "form-data; name=\"metadata\"")
	metaWriter, err := writer.CreatePart(mh)
	if err != nil {
		return progress, err
	}

	err = json.NewEncoder(metaWriter).Encode(meta)
	if err != nil {
		return progress, err
	}

	ch := make(textproto.MIMEHeader)
	ch.Set("Content-Type", "application/json")
	ch.Set("Content-Disposition", "form-data; name=\"checksums\"")
	metaWriter, err = writer.CreatePart(ch)
	if err != nil {
		return progress, err
	}

	_, err = metaWriter.Write([]byte("{}"))
	if err != nil {
		return progress, err
	}

	// Create a request
	// Cannot use client.Post here, as we need to set headers
	req, err := http.NewRequest("POST", wrapper.Endpoint+urlUploadBase+"/chunked/"+progress.Reference+"/complete", body)
	if err != nil {
		return progress, err
	}
	req = req.WithContext(ctx)
	//req.Header.Add("Content-Type", "multipart/form")

	// Make the request
	response, err := client.Do(req)
	if err != nil {
		return progress, err
	}

	// We have a response body if we get here, so make sure we close it when done
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return progress, fmt.Errorf(response.Status, response.StatusCode)
	}

	var fileMetadata FileMetadata

	err = json.NewDecoder(response.Body).Decode(&fileMetadata)
	if err != nil {
		return progress, err
	}

	progress.Reference = fileMetadata.DataStoreRef
	progress.Complete = true
	return progress, nil
}

func copyBufferWithCTX(ctx context.Context, dst io.Writer, src io.Reader, chunkSize int64) (hash string, err error) {
	md5er := md5.New() // #nosec
	var written int64
	teeReader := io.TeeReader(src, md5er)
	buffSize := copyBufferSize
	mut := &sync.Mutex{}
	ok := true
	currentContext, Done := context.WithCancel(ctx)
	isOk := true
	defer func() {
		if isOk {
			Done()
		}
	}()
	go func() {
		<-currentContext.Done()

		mut.Lock()
		defer mut.Unlock()
		ok = false
	}()
	var left int64
	left = chunkSize - written

	for left > 0 {

		mut.Lock()
		isOk = ok
		mut.Unlock()
		if !isOk {
			break
		}

		left = chunkSize - written
		if left < buffSize {
			buffSize = left
		}

		buf := make([]byte, buffSize)
		nr, er := teeReader.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			err = er
			break
		}
	}
	if !isOk {
		err = fmt.Errorf(errStreamInterrupted)
	}

	hash = hex.EncodeToString(md5er.Sum(nil))
	return hash, err
}
