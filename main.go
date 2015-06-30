package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-go/statsd"
)

var (
	metricPattern *regexp.Regexp

	// configurable values
	tags      = []string{}
	namespace string
	listen    string
)

func init() {
	metricPattern = regexp.MustCompile(`^[a-z][\.a-z0-9]*[a-z]$`)

	tagPattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*(:[a-zA-Z_0-9]*)?$`)
	for _, v := range strings.Split(os.Getenv("TAGS"), ",") {
		tag := strings.TrimSpace(v)
		if tag != "" && tagPattern.Match([]byte(tag)) == false {
			log.Fatalf("Invalid Tag: %s, must match: %s\n", tag, tagPattern.String())
		}
		tags = append(tags, tag)
	}

	namespacePattern := regexp.MustCompile(`^[a-z][a-z0-9\.]*[a-z0-9]\.$`)
	namespace = strings.TrimSpace(os.Getenv("NAMESPACE"))
	if namespace == "" {
		namespace = "experimental."
	}

	if namespacePattern.Match([]byte(namespace)) == false {
		log.Fatalf("Invalid namespace `%s`, must match: %s\n", namespace, namespacePattern.String())
	}

	listen = os.Getenv("LISTEN")
	if listen == "" {
		listen = ":8080"
	}
}

func metricNameOK(metricName string) bool {
	return metricPattern.Match([]byte(metricName))
}

func extractMetric(requestUrl *url.URL) (string, error) {
	parts := strings.Split(requestUrl.Path, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("No metric provided")
	}

	if !metricNameOK(parts[2]) {
		return "", fmt.Errorf("Invalid metric name")
	}

	return parts[2], nil
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
		} else {
			io.WriteString(w, "OK")
		}
	}
}

func main() {

	ddClient, err := statsd.New("127.0.0.1:8125")
	if err != nil {
		log.Fatal(err)
	}
	ddClient.Namespace = "experimental."

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

	fmt.Printf("Namespace	: %s\n", namespace)
	fmt.Printf("Tags		: %s\n", tags)
	fmt.Printf("Listen		: %v\n", listen)

	log.Fatal(http.ListenAndServe(listen, nil))

}
