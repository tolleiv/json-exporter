package main

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"fmt"
	"net/url"
	"io/ioutil"
)

var probeTests = []struct {
	in_data       string
	in_field      string
	out_http_code int
	out_value     string
}{
	{"{\"field\": 23}", "$.field", 200, "value 23"},
	{"{\"field\": 19}", "$.field", 200, "value 19 "},
}

func TestHTTP(t *testing.T) {

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
			t.Error("HTTP Code mismatch")
		}

		//		fmt.Println(resp.StatusCode)
		//		fmt.Println(resp.Header.Get("Content-Type"))
		fmt.Println(string(body))
	}
}
