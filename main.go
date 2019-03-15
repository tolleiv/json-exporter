package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yalp/jsonpath"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
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
		lookuppath = "$[\"oracle.cloudstorage.galaxy.mds.health.MdsRdbHealthCheck\"][\"ossstore.1.$CONFIG_SHARDS$.$DEFAULT_SHARD$\"][\"borrowedConnections\"]"
		log.Printf("setting jsonpath to default %s ", lookuppath)
	}
	probeSuccessGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_success",
		Help: "Displays whether or not the probe was a success",
	})
	probeDurationGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_duration_seconds",
		Help: "Returns how long the probe took to complete in seconds",
	})
	valueGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "value",
			Help: "Retrieved value",
		},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(probeSuccessGauge)
	registry.MustRegister(probeDurationGauge)
	registry.MustRegister(valueGauge)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	reqwithheader, err := http.NewRequest("GET", target, nil)
	if err != nil {
		log.Fatal(err)
	}
	reqwithheader.Header.Set("Accept", "application/json")
	resp, err := client.Do(reqwithheader)
	if err != nil {
		log.Fatal(err)

	} else {
		defer resp.Body.Close()
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var json_data map[string]interface{}
		if err := json.Unmarshal(bytes, &json_data); err != nil {
			fmt.Printf("while unmarshalling %v", bytes)
			panic(err)
		}
		Filter, err := jsonpath.Prepare(lookuppath)
		if err != nil {
			jpe := fmt.Sprintf("jsonpath Prepare %v", err)
			http.Error(w, jpe, http.StatusNotFound)
			return
		}
		out, err := Filter(json_data)
		if err != nil {
			fmt.Printf("%v", json_data)
			jpe := fmt.Sprintf("jsonpath on execute %v", err)
			http.Error(w, jpe, http.StatusNotFound)
			return
		}
		log.Printf("Found value %v", out)
		number, ok := out.(float64)
		if !ok {
			strout, _ := out.(string)
			number, err = strconv.ParseFloat(strout, 64)
			if err != nil {
				http.Error(w, "Values could not be parsed to Float64", http.StatusInternalServerError)
				return
			}
		}
		probeSuccessGauge.Set(1)
		valueGauge.Set(number)
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}
