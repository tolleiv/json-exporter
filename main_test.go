package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

var probeTests = []struct {
	in_data       string
	in_field      string
	out_http_code int
	out_value     string
}{
	{"{\"field\": 23}", "$.field", 200, "value 23"},
	{"{\"field\": 19}", "$.field", 200, "value 19"},
	{"{\"field\": 19}", "$.undefined", 404, "jsonpath on execute child 'undefined' not found in JSON object at 3"},
	{"{\"field.x.y\": 37}", "$[\"field.x.y\"]", 200, "value 37"},
}

func TestProbeHandler(t *testing.T) {

	for _, tt := range probeTests {

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, tt.in_data)
		}))
		defer ts.Close()

		u := fmt.Sprintf("http://example.com/probe?target=%s&jsonpath=%s", url.QueryEscape(ts.URL), url.QueryEscape(tt.in_field))

		req := httptest.NewRequest("GET", u, nil)
		w := httptest.NewRecorder()

		probeHandler(w, req)

		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		if tt.out_http_code != resp.StatusCode {
			t.Error(fmt.Sprintf("HTTP Code mismatch - %d expected %d", resp.StatusCode, tt.out_http_code))
		}

		if !strings.Contains(string(body), tt.out_value) {
			t.Error(fmt.Sprintf("Expected output: %s got\n%s", tt.out_value, string(body)))
		}
	}
}
