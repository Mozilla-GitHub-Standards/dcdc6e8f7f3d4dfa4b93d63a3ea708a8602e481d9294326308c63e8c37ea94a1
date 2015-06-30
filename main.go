package main

import (
	"bufio"
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

	// white listed metrics that are allowed to be sent
	whitelistPath string
	whitelist     = map[string]bool{}
)

func init() {
	metricPattern = regexp.MustCompile(`^[a-z][\.a-z0-9]*[a-z0-9]$`)

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

	whitelistPath = os.Getenv("WHITELIST_FILE")
	if whitelistPath != "" {
		file, err := os.Open(whitelistPath)
		if err != nil {
			log.Fatalf("Whitelist file error %s\n", err.Error())
		}

		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			metricName := strings.TrimSpace(scanner.Text())

			if string(metricName[0]) == "#" {
				continue
			}

			if metricNameOK(metricName) == false {
				log.Fatalf("Invalid metric name: %s\n", metricName)
			}

			whitelist[metricName] = true
		}
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
			io.WriteString(w, "Invalid metric name")
			return
		}

		if whitelistPath != "" && whitelist[metricName] != true {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, "Metric is not whitelisted")
			return
		}

		// handle the request and deal with any errors
		if err := handler(metricName, r); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "ERROR: "+err.Error())
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
	ddClient.Namespace = namespace

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

	log.Println("Starting service...")
	log.Println("--------------------------------------")
	log.Printf("Namespace : %s", namespace)
	log.Printf("Tags      : %s", tags)
	log.Printf("Listening : %v", listen)

	if whitelistPath != "" {
		log.Printf("Whitelisted metrics: ")
		for k, _ := range whitelist {
			log.Printf(" - %s ", k)
		}
	} else {
		log.Printf("Whitelist : <all allowed>")
	}

	log.Fatal(http.ListenAndServe(listen, nil))
}
