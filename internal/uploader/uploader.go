package uploader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	// API framework
	"github.com/ant0ine/go-json-rest/rest"
	// Validate input filetype
	"github.com/h2non/filetype"
	// Google Cloud client
	"cloud.google.com/go/storage"
)

var (
	supportedTypes = []string{"image/gif", "video/mp4", "video/webm", "video/x-msvideo"}
)

const (
	tmpPath      = "/tmp/"
	fileNameSize = 16
)

func NewUploader(cfg map[string]string) Uploader {
	fsize, err := strconv.ParseInt(cfg["GIFKA_UPLOADER_MAX_FILE_SIZE"], 10, 64)
	if err != nil {
		panic(err)
	}

	return Uploader{
		bucket:      cfg["GIFKA_UPLOADER_GC_BUCKET"],
		maxFileSize: fsize,
	}
}

type UploadError struct {
	code    int
	message string
}

type Uploader struct {
	bucket      string
	maxFileSize int64
}

func (u *Uploader) Upload(writer rest.ResponseWriter, req *rest.Request) {
	// Upload a file to the local disk
	req.ParseMultipartForm(u.maxFileSize << 20)
	fname, e := uploadToDisk(req)
	if e != nil {
		rest.Error(writer, e.message, e.code)
		return
	}

	// Upload a file to GCS
	objAttrs, err := uploadToGCS(fname, u.bucket)
	if err != nil {
		switch err {
		case storage.ErrBucketNotExist:
			rest.Error(writer, "Cannot upload the file to GCS", http.StatusInternalServerError)
		default:
			rest.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	}

	// Remove uploaded a file from the local disk
	err = os.Remove(tmpPath + fname)
	if err != nil {
		rest.Error(writer, err.Error(), http.StatusInternalServerError)
	}

	writer.WriteJson(map[string]string{"Body": objAttrs.Name})
}

func uploadToDisk(req *rest.Request) (string, *UploadError) {
	srcFile, srcAttrs, err := req.FormFile("file")
	if err != nil {
		return srcAttrs.Filename, &UploadError{http.StatusBadRequest, err.Error()}
	}
	defer srcFile.Close()

	// We only have to pass the file header = first 261 bytes
	head := make([]byte, 261)
	srcFile.Read(head)
	if !isValid(head) {
		return srcAttrs.Filename, &UploadError{http.StatusBadRequest, "Supported file types: mp4, gif, webm and avi"}
	}

	dstFile, err := os.OpenFile(tmpPath+srcAttrs.Filename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return srcAttrs.Filename, &UploadError{http.StatusInternalServerError, err.Error()}
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return srcAttrs.Filename, &UploadError{http.StatusInternalServerError, err.Error()}
	}

	return srcAttrs.Filename, nil
}

func uploadToGCS(name, bucketName string) (*storage.ObjectAttrs, error) {
	f, err := os.Open(tmpPath + name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	bh := client.Bucket(bucketName)
	// Next check if the bucket exists
	if _, err = bh.Attrs(ctx); err != nil {
		return nil, err
	}

	path := getBucketPath()
	genFileName, err := generateRandomString(fileNameSize)
	if err != nil {
		return nil, err
	}

	obj := bh.Object(path + genFileName + filepath.Ext(name))
	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, f); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	attrs, err := obj.Attrs(ctx)
	return attrs, err
}

func getBucketPath() string {
	t := time.Now()
	return fmt.Sprintf("media/%v/", t.Format("2006/01/02"))
}

func isValid(buf []byte) bool {
	kind, unknown := filetype.Match(buf)
	if unknown != nil {
		return false
	}

	valid := false

	for _, t := range supportedTypes {
		if t == kind.MIME.Value {
			valid = true
		}
	}

	return valid
}
