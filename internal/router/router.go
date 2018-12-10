package router

import (
	"log"

	// API framework
	"github.com/ant0ine/go-json-rest/rest"
	// Middlewares
	"github.com/gifkatv/converter/internal/middleware"
	// Uploader
	"github.com/gifkatv/converter/internal/uploader"
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
				u := uploader.NewUploader(cfg)
				u.Upload(w, r)
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
