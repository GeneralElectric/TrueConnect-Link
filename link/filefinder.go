package link

import (
	"crypto/sha256"
	"fmt"
	"github.com/GeneralElectric/TrueConnect-Link/trueconnect"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type foundFile struct {
	uri        string
	size       int64
	modifyTime time.Time
	hash       string
	target     *Target
	progress   trueconnect.UploadProgress
}

const (
	errTerminating = "Terminating"
)

func (linkClient *linkClient) findFiles(target Target, foundFiles *chan foundFile) error {
	err := filepath.Walk(target.Location, func(root string, info os.FileInfo, err error) error {
		if linkClient.isStopping {
			return io.EOF
		}
		if err == nil {
			if info.IsDir() {
				if target.Location == root {
					return nil
				}
				if target.Recursive == false {
					return filepath.SkipDir
				}
			}

			matched := false

			matched, err = regexp.MatchString(target.Match, root)
			if matched && (err == nil) {
				hash, err := computeSHA256Hash(root)
				if err != nil {
					return err
				}
				select {
				case *foundFiles <- foundFile{uri: root, size: info.Size(), target: &target, modifyTime: info.ModTime(), hash: hash}:
					break
				case <-linkClient.currentContext.Done():
					return fmt.Errorf(errTerminating)
				}
			}
		}

		return err
	})

	return err
}

func (foundFile *foundFile) getMetadata() map[string]trueconnect.MetadataValue {
	var meta map[string]trueconnect.MetadataValue
	if foundFile != nil && foundFile.uri != "" {
		meta = make(map[string]trueconnect.MetadataValue)
		meta[trueconnect.TenantID] = trueconnect.MetadataValue{Value: foundFile.target.Tenant, Immutable: true}
		meta[trueconnect.DataType] = trueconnect.MetadataValue{Value: foundFile.target.DataType, Immutable: true}
		meta[trueconnect.FileFormat] = trueconnect.MetadataValue{Value: foundFile.target.DataFormat, Immutable: true}
		meta[trueconnect.OriginalFileName] = trueconnect.MetadataValue{Value: foundFile.uri, Immutable: true}
		host, _ := os.Hostname()
		meta[sourceHost] = trueconnect.MetadataValue{Value: host, Immutable: true}
		meta[fileSize] = trueconnect.MetadataValue{Value: fmt.Sprintf("%v", foundFile.size), Immutable: true}
		meta[lastModifiedDate] = trueconnect.MetadataValue{Value: foundFile.modifyTime.Format(time.RFC3339), Immutable: true}
		meta[sha256Hash] = trueconnect.MetadataValue{Value: foundFile.hash, Immutable: true}
		for _, Lookup := range foundFile.target.PathEncodedMetaDataTags {
			regularExpression := regexp.MustCompile(Lookup.Match)
			matchedPatterns := regularExpression.FindStringSubmatch(foundFile.uri)
			if matchedPatterns != nil && len(matchedPatterns) > 1 {
				meta[Lookup.Tag] = trueconnect.MetadataValue{Value: matchedPatterns[1], Immutable: false}
			}
		}

		for _, staticTag := range foundFile.target.StaticTags {
			meta[staticTag.Tag] = trueconnect.MetadataValue{Value: staticTag.Value, Immutable: true}
		}
	}

	return meta
}

func computeSHA256Hash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
