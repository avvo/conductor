package conductor

import (
  "fmt"
  log "github.com/Sirupsen/logrus"
  "html"
  "net/http"
)

func noMatchingMountPointHandler(w http.ResponseWriter, r *http.Request) {
  log.WithFields(log.Fields{"url": r.URL.Path,
    "remote_address": r.RemoteAddr,
    "error":          "no_matching_mount_point",
  }).Warn("No mount point matches")
  http.Error(w,
    fmt.Sprintf(`{"error":"no_matching_mount_point","message":"I have no backend servers that handle '%s'"}`,
      html.EscapeString(r.URL.Path)), http.StatusBadGateway)
}

func noHealthyBackends(w http.ResponseWriter, r *http.Request) {
  log.WithFields(log.Fields{"url": r.URL.Path,
    "remote_address": r.RemoteAddr,
    "error":          "no_health_backends",
  }).Warn("No healthy backends")
  http.Error(w,
    fmt.Sprintf(`{"error":"no_healthy_backends","message":"There are no healthy backends that handle '%s'"}`,
      html.EscapeString(r.URL.Path)), http.StatusServiceUnavailable)
}

// Simply sends a 204, No content
func pingHandler(w http.ResponseWriter, r *http.Request) {
  w.WriteHeader(http.StatusNoContent)
}
