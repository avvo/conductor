package conductor

import (
  "io/ioutil"
  "net/http"
  "net/http/httptest"
  "testing"
)

func init() {

}

func TestPingHandler(t *testing.T) {
  ts := httptest.NewServer(http.HandlerFunc(pingHandler))
  defer ts.Close()

  res, err := http.Get(ts.URL)
  if err != nil {
    t.Fatal(err)
  }
  response, err := ioutil.ReadAll(res.Body)
  res.Body.Close()
  if err != nil {
    t.Fatal(err)
  }

  if res.Status != "204 No Content" {
    t.Errorf("Expected response status code to be 204 No Content, got '%v'", res.Status)
  }

  if string(response) != "" {
    t.Errorf("Expected response to be: '', but got '%s'", response)
  }
}
