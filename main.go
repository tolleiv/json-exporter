package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus"
	"crypto/tls"
	"github.com/oliveagle/jsonpath"
	"io/ioutil"
	"encoding/json"
)

var addr = flag.String("listen-address", ":9116", "The address to listen on for HTTP requests.")

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head><title>Json Exporter</title></head>
            <body>
            <h1>Json Exporter</h1>
            <p><a href="/probe">Run a probe</a></p>
            <p><a href="/metrics">Metrics</a></p>
            </body>
            </html>`))
	})
	flag.Parse()
	http.HandleFunc("/probe", probeHandler)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func probeHandler(w http.ResponseWriter, r *http.Request) {

	get_params := r.URL.Query()
	a_param := make(map[string]string)

	// log.Printf("get_params: %v", get_params)
	for k, v := range get_params {
		log.Printf("key[%s] value %s\n", k, v)
		if( strings.Contains(k , "jsonpath.") ) {
			a_param[strings.Replace(k,"jsonpath.","",-1)] = get_params.Get(k)
		}
	}
	// log.Printf("a_target: %v", a_target)

	target := get_params.Get("target")
	if target == "" {
		http.Error(w, "Target parameter is missing", 400)
		return
	}
	if (len(a_param) == 0){
		http.Error(w, "No JsonPath to lookup", 400)
		return
	}
	probeSuccessGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_success",
		Help: "Displays whether or not the probe was a success",
	})
	probeDurationGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_duration_seconds",
		Help: "Returns how long the probe took to complete in seconds",
	})

	registry := prometheus.NewRegistry()
	registry.MustRegister(probeSuccessGauge)
	registry.MustRegister(probeDurationGauge)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(target)
	if err != nil {
		log.Fatal(err)

	} else {
		defer resp.Body.Close()
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var json_data interface{}
		json.Unmarshal([]byte(bytes), &json_data)

	   for metric_name, json_path := range a_param {
			res, err := jsonpath.JsonPathLookup(json_data, json_path)
			if err != nil {
				http.Error(w, "Jsonpath not found", http.StatusNotFound)
				return
			}
			log.Printf("Found value %v for path %s", res, json_path)
			number, ok := res.(float64)
			if !ok {
				http.Error(w, "Values could not be parsed to Float64", http.StatusInternalServerError)
				return
			}
			valueGauge := prometheus.NewGauge(prometheus.GaugeOpts{
				Name:	metric_name,
				Help:	"Retrieved value",
			})
			registry.MustRegister(valueGauge)
			valueGauge.Set(number)
		}
		probeSuccessGauge.Set(1)
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}
