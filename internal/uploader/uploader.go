package uploader

import (
	"context"
	"net/http"
	"io"
	"os"
	"strconv"

	// API framework
	"github.com/ant0ine/go-json-rest/rest"

	// Validate input filetype
	"github.com/h2non/filetype"

	// Google Cloud client
	"cloud.google.com/go/storage"
)

func NewUploader(cfg map[string]string) Uploader {
	fsize, err := strconv.ParseInt(cfg["GIFKA_UPLOADER_MAX_FILE_SIZE"], 10, 64)
	if err != nil {
		panic(err)
	}

	return Uploader{
		bucket: cfg["GIFKA_UPLOADER_GC_BUCKET"],
		maxFileSize: fsize,
	}
}

var supportedTypes = []string{"image/gif", "video/mp4", "video/webm", "video/x-msvideo"}

type Uploader struct {
	bucket string
	maxFileSize int64
}

func (u *Uploader) Upload(writer rest.ResponseWriter, request *rest.Request) {
	request.ParseMultipartForm(u.maxFileSize << 20)
	formFile, fheader, err := request.FormFile("file")
	if err != nil {
		rest.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	defer formFile.Close()

	// We only have to pass the file header = first 261 bytes
	head := make([]byte, 261)
	formFile.Read(head)
	if !isValid(head) {
		rest.Error(writer, "Supported file types: mp4, gif, webm and avi", http.StatusBadRequest)
		return
	}

	fh, err := os.OpenFile("/tmp/" + fheader.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		rest.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer fh.Close()

	if _, err := io.Copy(fh, formFile); err != nil {
		rest.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	objAttrs, err := uploadToGoogleStorage(fh, u.bucket, fheader.Filename)
	if err != nil {
		switch err {
		case storage.ErrBucketNotExist:
			rest.Error(writer, "Cannot upload the file to GCS", http.StatusInternalServerError)
		default:
			rest.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	}

	writer.WriteJson(map[string]string{"Body": objAttrs.Name})

	// 	_, err := enqueuer.Enqueue("send_email", work.Q{"address": "test@example.com", "subject": "hello world", "customer_id": 4})
	// 	if err != nil {
	// 	 log.Fatal(err)
	// 	}
	// 	Process an uploaded file in a background
	// },
	// }
}

func uploadToGoogleStorage(r io.Reader, bucketName, name string) (*storage.ObjectAttrs, error) {
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

	obj := bh.Object(name)
	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	attrs, err := obj.Attrs(ctx)
	return attrs, err
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
