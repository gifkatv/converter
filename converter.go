package main

import (
	"os"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/coreos/go-semver/semver"
)

type SemVerMiddleware struct {
	MinVersion string
	MaxVersion string
}

func (mw *SemVerMiddleware) MiddlewareFunc(handler rest.HandlerFunc) rest.HandlerFunc {
	minVersion, err := semver.NewVersion(mw.MinVersion)
	if err != nil {
		panic(err)
	}

	maxVersion, err := semver.NewVersion(mw.MaxVersion)
	if err != nil {
		panic(err)
	}

	return func(writer rest.ResponseWriter, request *rest.Request) {
		version, err := semver.NewVersion(request.PathParam("version"))
		if err != nil {
			rest.Error(
				writer,
				"Invalid version: "+err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		if version.LessThan(*minVersion) {
			rest.Error(
				writer,
				"Min supported version is "+minVersion.String(),
				http.StatusBadRequest,
			)
			return
		}

		if maxVersion.LessThan(*version) {
			rest.Error(
				writer,
				"Max supported version is "+maxVersion.String(),
				http.StatusBadRequest,
			)
			return
		}

		request.Env["VERSION"] = version
		handler(writer, request)
	}
}

func loadEnvironment() map[string]string {
	env := os.Getenv("GIFKA_ENV")
	if "" == env {
		env = "development"
	}

	var environment map[string]string
	environment, err := godotenv.Read(".env." + env)

	if (err != nil) {
		panic(err)
	}

	return environment
}

func main() {
	env := loadEnvironment()

	// svmw := SemVerMiddleware{
	// 	MinVersion: "1.0.0",
	// 	MaxVersion: "1.0.0",
	// }

	statusMiddleware := &rest.StatusMiddleware{}
	api := rest.NewApi()
	api.Use(statusMiddleware)
	api.Use(rest.DefaultDevStack...)

	auth := &rest.AuthBasicMiddleware{
		Realm: "Converter API",
		Authenticator: func(username string, password string) bool {
			if username == env["GIFKA_API_STATUS_USERNAME"] && password == env["GIFKA_API_STATUS_PASSWORD"] {
				return true
			}
			return false
		},
	}

	router, err := rest.MakeRouter(
		rest.Get("/.status", auth.MiddlewareFunc(
			func(writer rest.ResponseWriter, _req *rest.Request) {
				writer.WriteJson(statusMiddleware.GetStatus())
			},
		)),
	)

	if err != nil {
		log.Fatal(err)
	}

	api.SetApp(router)
	http.Handle("/api/", http.StripPrefix("/api", api.MakeHandler()))
	log.Fatal(http.ListenAndServe(":" + env["GIFKA_PORT"], nil))
}
