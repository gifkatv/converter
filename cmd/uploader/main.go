package main

import (
  "log"
  "net/http"

  // Local packages
  "github.com/gifkatv/converter/config"
  apiV1 "github.com/gifkatv/converter/internal/api/v1"
)

func main() {
  cfg := config.Load()
  api := apiV1.New(cfg)

  http.Handle("/api/", http.StripPrefix("/api", api.MakeHandler()))
  log.Fatal(http.ListenAndServe(":" + cfg["GIFKA_UPLOADER_PORT"], nil))
}
