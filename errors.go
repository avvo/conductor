package conductor

type NoHealthyNodesError struct {
  Message     string
  ServiceName string
  Path        string
}

func NewNoHealthyNodesError(service, path string) *NoHealthyNodesError {
  return &NoHealthyNodesError{
    Message:     "No healthy nodes found",
    ServiceName: service,
    Path:        path,
  }
}

func (e *NoHealthyNodesError) Error() string {
  return e.Message
}

type NoMatchingMountPointError struct {
  Message string
  Path    string
}

func NewNoMatchingMountPointError(path string) *NoMatchingMountPointError {
  return &NoMatchingMountPointError{
    Message: "No mount point matches",
    Path:    path,
  }
}

func (e *NoMatchingMountPointError) Error() string {
  return e.Message
}
