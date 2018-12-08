package apiV1

import (
	// API framework
	"github.com/ant0ine/go-json-rest/rest"

	// Router
	"github.com/gifkatv/converter/internal/router"

	// Validate input filetype
	// "github.com/h2non/filetype"

	// "cloud.google.com/go/storage"

	// Background jobs
	// "github.com/gomodule/redigo/redis"
	// "github.com/gocraft/work"
)

// const RedisNamespace = "gifka"




// Make a redis pool
// var redisPool = &redis.Pool{
// 	MaxActive: 5,
// 	MaxIdle: 5,
// 	Wait: true,
// 	Dial: func() (redis.Conn, error) {
// 		return redis.Dial("tcp", ":6379")
// 	},
// }

// // Make an enqueuer with a particular namespace
// var enqueuer = work.NewEnqueuer(RedisNamespace, redisPool)

func New(cfg map[string]string) *rest.Api {
	statusMiddleware := &rest.StatusMiddleware{}
	api := rest.NewApi()
	api.Use(statusMiddleware)
	api.Use(rest.DefaultDevStack...)

	api.SetApp(router.NewRouter(cfg, statusMiddleware))

	return api
}
