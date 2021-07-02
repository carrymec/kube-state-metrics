package main

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_model/go"
)

const magicString = "zZgWfBxLqvG8kc8IMv3POi2Bb0tZI3vAnBx+gBaFi9FyPzB/CzKUer1yufDa"

const (
	closeNotifier = 1 << iota
	flusher
	hijacker
	readerFrom
	pusher
)

var pickDelegator = make([]func(*responseWriterDelegator) delegator, 32)

type delegator interface {
	http.ResponseWriter

	Status() int
	Written() int64
}

type responseWriterDelegator struct {
	http.ResponseWriter

	status             int
	written            int64
	wroteHeader        bool
	observeWriteHeader func(int)
}

var emptyLabels = prometheus.Labels{}

func InstrumentHandlerDuration(obs prometheus.ObserverVec, next http.Handler) http.HandlerFunc {
	code, method := checkLabels(obs)

	if code {
		return func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			d := newDelegator(w, nil)
			next.ServeHTTP(d, r)

			obs.With(labels(code, method, r.Method, d.Status())).Observe(time.Since(now).Seconds())
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		next.ServeHTTP(w, r)
		obs.With(labels(code, method, r.Method, 0)).Observe(time.Since(now).Seconds())
	}
}

func checkLabels(c prometheus.Collector) (code bool, method bool) {
	// TODO(beorn7): Remove this hacky way to check for instance labels
	// once Descriptors can have their dimensionality queried.
	var (
		desc *prometheus.Desc
		m    prometheus.Metric
		pm   io_prometheus_client.Metric
		lvs  []string
	)

	// Get the Desc from the Collector.
	descc := make(chan *prometheus.Desc, 1)
	c.Describe(descc)

	select {
	case desc = <-descc:
	default:
		panic("no description provided by collector")
	}
	select {
	case <-descc:
		panic("more than one description provided by collector")
	default:
	}

	close(descc)

	// Make sure the Collector has a valid Desc by registering it with a
	// temporary registry.
	prometheus.NewRegistry().MustRegister(c)

	// Create a ConstMetric with the Desc. Since we don't know how many
	// variable labels there are, try for as long as it needs.
	for err := errors.New("dummy"); err != nil; lvs = append(lvs, magicString) {
		m, err = prometheus.NewConstMetric(desc, prometheus.UntypedValue, 0, lvs...)
	}

	// Write out the metric into a proto message and look at the labels.
	// If the value is not the magicString, it is a constLabel, which doesn't interest us.
	// If the label is curried, it doesn't interest us.
	// In all other cases, only "code" or "method" is allowed.
	if err := m.Write(&pm); err != nil {
		panic("error checking metric for labels")
	}
	for _, label := range pm.Label {
		name, value := label.GetName(), label.GetValue()
		if value != magicString || isLabelCurried(c, name) {
			continue
		}
		switch name {
		case "code":
			code = true
		case "method":
			method = true
		default:
			panic("metric partitioned with non-supported labels")
		}
	}
	return
}

func isLabelCurried(c prometheus.Collector, label string) bool {
	switch v := c.(type) {
	case *prometheus.CounterVec:
		if _, err := v.CurryWith(prometheus.Labels{label: "dummy"}); err == nil {
			return false
		}
	case prometheus.ObserverVec:
		if _, err := v.CurryWith(prometheus.Labels{label: "dummy"}); err == nil {
			return false
		}
	default:
		panic("unsupported metric vec type")
	}
	return true
}

func newDelegator(w http.ResponseWriter, observeWriteHeaderFunc func(int)) delegator {
	d := &responseWriterDelegator{
		ResponseWriter:     w,
		observeWriteHeader: observeWriteHeaderFunc,
	}

	id := 0
	//nolint:staticcheck // Ignore SA1019. http.CloseNotifier is deprecated but we keep it here to not break existing users.
	if _, ok := w.(http.CloseNotifier); ok {
		id += closeNotifier
	}
	if _, ok := w.(http.Flusher); ok {
		id += flusher
	}
	if _, ok := w.(http.Hijacker); ok {
		id += hijacker
	}
	if _, ok := w.(io.ReaderFrom); ok {
		id += readerFrom
	}
	if _, ok := w.(http.Pusher); ok {
		id += pusher
	}

	return pickDelegator[id](d)
}

func labels(code, method bool, reqMethod string, status int) prometheus.Labels {
	if !(code || method) {
		return emptyLabels
	}
	labels := prometheus.Labels{}

	if code {
		labels["code"] = sanitizeCode(status)
	}
	if method {
		labels["method"] = sanitizeMethod(reqMethod)
	}

	return labels
}

func sanitizeMethod(m string) string {
	switch m {
	case "GET", "get":
		return "get"
	case "PUT", "put":
		return "put"
	case "HEAD", "head":
		return "head"
	case "POST", "post":
		return "post"
	case "DELETE", "delete":
		return "delete"
	case "CONNECT", "connect":
		return "connect"
	case "OPTIONS", "options":
		return "options"
	case "NOTIFY", "notify":
		return "notify"
	default:
		return strings.ToLower(m)
	}
}

func sanitizeCode(s int) string {
	switch s {
	case 100:
		return "100"
	case 101:
		return "101"

	case 200, 0:
		return "200"
	case 201:
		return "201"
	case 202:
		return "202"
	case 203:
		return "203"
	case 204:
		return "204"
	case 205:
		return "205"
	case 206:
		return "206"

	case 300:
		return "300"
	case 301:
		return "301"
	case 302:
		return "302"
	case 304:
		return "304"
	case 305:
		return "305"
	case 307:
		return "307"

	case 400:
		return "400"
	case 401:
		return "401"
	case 402:
		return "402"
	case 403:
		return "403"
	case 404:
		return "404"
	case 405:
		return "405"
	case 406:
		return "406"
	case 407:
		return "407"
	case 408:
		return "408"
	case 409:
		return "409"
	case 410:
		return "410"
	case 411:
		return "411"
	case 412:
		return "412"
	case 413:
		return "413"
	case 414:
		return "414"
	case 415:
		return "415"
	case 416:
		return "416"
	case 417:
		return "417"
	case 418:
		return "418"

	case 500:
		return "500"
	case 501:
		return "501"
	case 502:
		return "502"
	case 503:
		return "503"
	case 504:
		return "504"
	case 505:
		return "505"

	case 428:
		return "428"
	case 429:
		return "429"
	case 431:
		return "431"
	case 511:
		return "511"

	default:
		return strconv.Itoa(s)
	}
}
