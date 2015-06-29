package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-go/statsd"
)

var metricPattern *regexp.Regexp

func init() {
	metricPattern = regexp.MustCompile(`^[a-z][\.a-z0-9]*[a-z]$`)
}

func extractMetric(requestUrl *url.URL) (string, error) {

	parts := strings.Split(requestUrl.Path, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("No metric provided")
	}

	metricName := parts[2]
	// check that metric name matches a pattern
	if metricPattern.Match([]byte(metricName)) == false {
		return "", fmt.Errorf("does not match pattern")
	}

	return metricName, nil
}

func getString(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", nil
	}

	b, err := ioutil.ReadAll(r.Body)

	if err != nil {
		return "", err
	}

	return string(b), nil
}

func getFloat64(req *http.Request) (float64, error) {
	s, err := getString(req)

	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(s, 64)
}

func getInt64(req *http.Request) (v int64, err error) {
	s, err := getString(req)
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(s, 10, 64)
}

type statHandler func(m string, r *http.Request) error

func makeHandler(handler statHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricName, err := extractMetric(r.URL)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			bytes.NewBufferString("Invalid metric name").WriteTo(w)
			return
		}

		// handle the request and deal with any errors
		if err := handler(metricName, r); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			bytes.NewBufferString("ERROR: " + err.Error() + "\n").WriteTo(w)
		}
	}
}

func main() {

	ddClient, err := statsd.New("127.0.0.1:8125")
	if err != nil {
		log.Fatal(err)
	}
	ddClient.Namespace = "experimental."

	tags := []string{
		"env:dev",
	}

	http.HandleFunc("/gauge/", makeHandler(func(m string, r *http.Request) error {
		val, err := getFloat64(r)
		if err != nil {
			return err
		}
		return ddClient.Gauge(m, val, tags, 1)
	}))

	http.HandleFunc("/count/", makeHandler(func(m string, r *http.Request) error {
		val, err := getInt64(r)
		if err != nil {
			return err
		}
		return ddClient.Count(m, val, tags, 1)
	}))

	http.HandleFunc("/histogram/", makeHandler(func(m string, r *http.Request) error {
		val, err := getFloat64(r)
		if err != nil {
			return err
		}
		return ddClient.Histogram(m, val, tags, 1)
	}))

	http.HandleFunc("/set/", makeHandler(func(m string, r *http.Request) error {
		val, err := getString(r)
		if err != nil {
			return err
		}
		return ddClient.Set(m, val, tags, 1)
	}))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
