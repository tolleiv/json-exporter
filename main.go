package main

import (
	"flag"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"crypto/tls"
	"encoding/json"
	"github.com/oliveagle/jsonpath"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
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

	params := r.URL.Query()
	target := params.Get("target")
	if target == "" {
		http.Error(w, "Target parameter is missing", 400)
		return
	}
	lookuppath := params.Get("jsonpath")
	if lookuppath == "" {
		http.Error(w, "The JsonPath to lookup", 400)
		return
	}
	multipleStr := params.Get("multiple")
	multiple, err := strconv.ParseFloat(multipleStr, 64)
	if err != nil {
		multiple = 1
	}
	labelPaths := params["label"]
	var labelKeys []string
	for _, labelPath := range labelPaths {
		labelParts := strings.Split(labelPath, ".")
		labelKeys = append(labelKeys, labelParts[len(labelParts)-1])
	}
	probeSuccessGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_success",
		Help: "Displays whether or not the probe was a success",
	})
	probeDurationGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_duration_seconds",
		Help: "Returns how long the probe took to complete in seconds",
	})
	valueGaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "value",
			Help: "Retrieved value",
		},
		labelKeys,
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(probeSuccessGauge)
	registry.MustRegister(probeDurationGauge)
	registry.MustRegister(valueGaugeVec)

	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(target)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		http.Error(w, "Target is irresponsible", http.StatusInternalServerError)
		log.Printf("Access to %v failed: %v", target, err)
		return
	} else {
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var json_data interface{}
		json.Unmarshal([]byte(bytes), &json_data)

		// Parse labels
		labels := make(prometheus.Labels)
		for idx, labelPath := range labelPaths {
			labelKey := labelKeys[idx]
			res, err := jsonpath.JsonPathLookup(json_data, labelPath)
			if err != nil {
				http.Error(w, "Jsonpath for label not found", http.StatusNotFound)
				log.Printf("Jsonpath for label(%v) not found: %v", labelKey, json_data)
				return
			}
			if str, ok := res.(string); ok {
				labels[labelKey] = str
			} else if number, ok := res.(float64); ok {
				labels[labelKey] = strconv.FormatFloat(number, 'f', -1, 64)
			} else if boolean, ok := res.(bool); ok {
				if boolean {
					labels[labelKey] = "true"
				} else {
					labels[labelKey] = "false"
				}
			} else {
				http.Error(w, "Failed to parse label", http.StatusInternalServerError)
				log.Printf("Failed to parse label: %v(%v)", res, reflect.TypeOf(res))
				return
			}
		}
		valueGauge := valueGaugeVec.With(labels)

		// Parse value
		res, err := jsonpath.JsonPathLookup(json_data, lookuppath)
		if err != nil {
			http.Error(w, "Jsonpath not found", http.StatusNotFound)
			log.Printf("Jsonpath(%v) not found: %v", lookuppath, json_data)
			return
		}
		//log.Printf("Found value %v", res)
		if number, ok := res.(float64); ok {
			probeSuccessGauge.Set(1)
			valueGauge.Set(number * multiple)
		} else if boolean, ok := res.(bool); ok {
			probeSuccessGauge.Set(1)
			if boolean {
				valueGauge.Set(1)
			} else {
				valueGauge.Set(0)
			}
		} else if str, ok := res.(string); ok {
			number, err = strconv.ParseFloat(str, 64)
			if err != nil {
				http.Error(w, "values is string but cannot be converted to Float64", http.StatusInternalServerError)
				log.Printf("%v(%v) could not be parsed to Float64", res, reflect.TypeOf(res))
			}
			probeSuccessGauge.Set(1)
			valueGauge.Set(number * multiple)
		} else if slice, ok := res.([]interface{}); ok {
			probeSuccessGauge.Set(1)
			valueGauge.Set(float64(len(slice)))
		} else {
			http.Error(w, "Values could not be parsed to Float64", http.StatusInternalServerError)
			log.Printf("%v(%v) could not be parsed to Float64", res, reflect.TypeOf(res))
			return
		}
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}
