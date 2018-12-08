package router

import (
	"io"
	"log"
	"net/http"
	"os"

	// API framework
	"github.com/ant0ine/go-json-rest/rest"
	// Validate input filetype
	"github.com/h2non/filetype"

	// Middlewares
	"github.com/gifkatv/converter/internal/middleware"
)

func NewRouter(cfg map[string]string, status *rest.StatusMiddleware) rest.App {
	return initRouter(cfg, status)
}

func initRouter(cfg map[string]string, status *rest.StatusMiddleware) rest.App {
	svmw := middleware.NewSemVerMiddleware()

	auth := &rest.AuthBasicMiddleware{
		Realm: "Converter API",
		Authenticator: func(username string, password string) bool {
			if username == cfg["GIFKA_API_STATUS_USERNAME"] && password == cfg["GIFKA_API_STATUS_PASSWORD"] {
				return true
			}

			return false
		},
	}

	router, err := rest.MakeRouter(
		rest.Get("/.status", auth.MiddlewareFunc(
			func(w rest.ResponseWriter, _r *rest.Request) {
				w.WriteJson(status.GetStatus())
			},
		)),

		rest.Post("/v#version/video", svmw.MiddlewareFunc(
			func(w rest.ResponseWriter, r *rest.Request) {
				r.ParseMultipartForm(64 << 20) // 64Mb
				file, handler, err := r.FormFile("file")

				if err != nil {
					rest.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				defer file.Close()

				tmpFile, err := os.OpenFile("/tmp/" + handler.Filename, os.O_WRONLY|os.O_CREATE, 0600)
				if err != nil {
					rest.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				// We only have to pass the file header = first 261 bytes
				head := make([]byte, 261)
				tmpFile.Read(head)

				if !(filetype.IsSupported("mp4") || filetype.IsSupported("gif") || filetype.IsSupported("webm") || filetype.IsSupported("avi")) {
					rest.Error(w, "Supported file types: mp4, gif, webm and avi", http.StatusBadRequest)
					return
				}

				defer tmpFile.Close()
				if _, err := io.Copy(tmpFile, file); err != nil {
					rest.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				// _, err := enqueuer.Enqueue("send_email", work.Q{"address": "test@example.com", "subject": "hello world", "customer_id": 4})
				// if err != nil {
				//  log.Fatal(err)
				// }
				// Process an uploaded file in a background
			},
		)),

		rest.Post("/v#version/video/batch", svmw.MiddlewareFunc(
			func(w rest.ResponseWriter, r *rest.Request) {
				// Process the list of uploaded files in a background
			},
		)),
	)

	if err != nil {
		log.Fatal(err)
	}

	return router
}
