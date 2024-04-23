package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny/api"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// ResultsToFields maps ExtractResult from ExtractAll to fields in a struct.
func ResultsToFields(result ExtractResult, fieldsToSet map[string]any) error {
	for key, fieldPtr := range fieldsToSet {
		if v, ok := result[key]; ok {
			err := CastStringToType(v, fieldPtr)
			if err != nil {
				return fmt.Errorf("error casting %s to type: %w", key, err)
			}
		}
	}
	return nil
}

// CastStringToType dynamically casts a string to a field's type and assigns it.
func CastStringToType(s string, fieldPtr any) error {
	if s == "" || s == "null" {
		return nil
	}
	if fieldPtr == nil {
		return errors.New("fieldPtr is nil")
	}
	switch f := fieldPtr.(type) {
	case **string:
		*f = &s
	case *string:
		*f = s
	case **int:
		i, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		*f = &i
	case *int:
		i, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		*f = i
	case **int64:
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*f = &i
	case *int64:
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*f = i
	case **float64:
		fl, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		*f = &fl
	case *float64:
		fl, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		*f = fl
	case **bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		*f = &b
	case *bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		*f = b
	default:
		return json.Unmarshal([]byte(s), fieldPtr)
	}
	return nil
}

func HasTxtFile(submission api.Submission) bool {
	for _, file := range submission.Files {
		if strings.HasPrefix(file.MimeType, "text") {
			return true
		}
	}
	return false
}

func HasJsonFile(submission api.Submission) bool {
	for _, file := range submission.Files {
		if strings.HasSuffix(file.MimeType, "json") {
			return true
		}
	}
	return false
}

func HasMetadata(submission api.Submission) bool {
	for _, file := range submission.Files {
		if strings.HasPrefix(file.MimeType, "text") {
			return true
		}
		if strings.HasSuffix(file.MimeType, "json") {
			return true
		}
	}
	return false
}

func FilterMetadata(submission api.Submission) (files []api.File) {
	for _, file := range submission.Files {
		if strings.HasPrefix(file.MimeType, "text") {
			files = append(files, file)
		}
		if strings.HasSuffix(file.MimeType, "json") {
			files = append(files, file)
		}
	}
	return files
}

type MetadataContent struct {
	Blob []byte
	api.File
}

func GetMetadataBytes(submission api.Submission) ([]MetadataContent, error) {
	var metadata []MetadataContent
	for _, file := range submission.Files {
		if !strings.HasPrefix(file.MimeType, "text") && !strings.HasSuffix(file.MimeType, "json") {
			continue
		}
		if file.FileURLFull == "" {
			continue
		}
		r, err := http.Get(file.FileURLFull)
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()
		if r.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %v", r.Status)
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		metadata = append(metadata, MetadataContent{b, file})
	}
	return metadata, nil
}
