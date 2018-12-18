package uploader

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
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

type Uploader struct {
	bucket      string
	maxFormSize int64
}

type UploadStatus struct {
	fname string
	code  int
	msg   string
}

func NewUploader(cfg map[string]string) Uploader {
	fsize, err := strconv.ParseInt(cfg["GIFKA_UPLOADER_MAX_FILE_SIZE"], 10, 64)
	if err != nil {
		panic(err)
	}

	filesCount, err := strconv.ParseInt(cfg["GIFKA_UPLOADER_FILES_COUNT"], 10, 64)
	if err != nil {
		panic(err)
	}

	return Uploader{
		bucket:      cfg["GIFKA_UPLOADER_GC_BUCKET"],
		maxFormSize: fsize * filesCount,
	}
}

func (u *Uploader) Upload(writer rest.ResponseWriter, req *rest.Request) {
	err := req.ParseMultipartForm(u.maxFormSize << 20)
	if err != nil {
		rest.Error(writer, err.Error(), http.StatusInternalServerError)
	}

	form := req.MultipartForm
	status := uploadToDisk(form.File["file"][0])
	if status.code >= 300 {
		rest.Error(writer, status.msg, status.code)
		return
	}

	// Upload a file to GCS
	objAttrs, err := uploadToGCS(status.fname, u.bucket)
	if err != nil {
		switch err {
		case storage.ErrBucketNotExist:
			rest.Error(writer, "Cannot upload the file to GCS", http.StatusInternalServerError)
		default:
			rest.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	}

	// Remove uploaded a file from the local disk
	err = os.Remove(tmpPath + status.fname)
	if err != nil {
		rest.Error(writer, err.Error(), http.StatusInternalServerError)
	}

	writer.WriteJson(map[string]string{"Body": objAttrs.Name})
}

func (u *Uploader) UploadBatch(writer rest.ResponseWriter, req *rest.Request) {
	err := req.ParseMultipartForm(u.maxFormSize << 20)
	if err != nil {
		rest.Error(writer, err.Error(), http.StatusInternalServerError)
	}

	form := req.MultipartForm
	files := form.File["files"]

	for _, f := range files {
		status := uploadToDisk(f)
		if status.code >= 300 {
			rest.Error(writer, status.msg, status.code)
			return
		}

		// Upload a file to GCS
		_, err := uploadToGCS(status.fname, u.bucket)
		if err != nil {
			switch err {
			case storage.ErrBucketNotExist:
				rest.Error(writer, "Cannot upload the file to GCS", http.StatusInternalServerError)
			default:
				rest.Error(writer, err.Error(), http.StatusInternalServerError)
			}
		}

		// Remove uploaded a file from the local disk
		err = os.Remove(tmpPath + status.fname)
		if err != nil {
			rest.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	}

	writer.WriteJson(map[string]string{"Body": "All good"})
}

func uploadToDisk(fh *multipart.FileHeader) UploadStatus {
	srcFile, err := fh.Open()
	if err != nil {
		return UploadStatus{fh.Filename, http.StatusBadRequest, err.Error()}
	}
	defer srcFile.Close()

	// We only have to pass the file header = first 261 bytes
	head := make([]byte, 261)
	srcFile.Read(head)
	if !isValid(head) {
		return UploadStatus{fh.Filename, http.StatusBadRequest, "Supported file types: mp4, gif, webm and avi"}
	}

	dstFile, err := os.OpenFile(tmpPath+fh.Filename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return UploadStatus{fh.Filename, http.StatusInternalServerError, err.Error()}
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return UploadStatus{fh.Filename, http.StatusInternalServerError, err.Error()}
	}

	return UploadStatus{fname: fh.Filename}
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
