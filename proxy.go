package gateway

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

type key int

const (
	requestIDKey key = 0
)

// Matcher is used for dynamic routing.
type Matcher func(uri, host string) string

// Proxy multiplexes w/ a pool of connections to the respective upstream.
type Proxy struct {
	mu            sync.Mutex
	Scheduler     *Scheduler
	Stats         *Stats
	log           *log.Logger
	conns         map[string]map[*httputil.ReverseProxy]struct{}
	routeMap      map[string]string
	defaultStatus int
	defaultBody   string
}

func (p *Proxy) SetDefaultResponse(statusCode int, body string) {
	p.defaultStatus = statusCode
	p.defaultBody = body
}

// NewProxy returns a new proxy.
func NewProxy(routes []Route, log *log.Logger) *Proxy {
	routeMap := map[string]string{}
	for _, v := range routes {
		routeMap[v.Backend] = v.PathPrefix
	}

	return &Proxy{
		Stats:         NewStats(),
		log:           log,
		conns:         make(map[string]map[*httputil.ReverseProxy]struct{}),
		routeMap:      routeMap,
		defaultBody:   "- not found -",
		defaultStatus: 400,
	}
}

type RequestCount struct {
	Success int32 `json:"success"`
	Error   int32 `json:"error"`
}

type LatencyMS struct {
	Average int64 `json:"average"`
	P99     int64 `json:"p99"`
}

type StatsResponse struct {
	RequestCount `json:"request_count"`
	LatencyMS    `json:"latency_ms"`
}

type stats struct{ s *Stats }

func (t *stats) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	success := t.s.Success()
	error := t.s.Error()
	p99 := t.s.P99()
	out := StatsResponse{
		RequestCount: RequestCount{
			Success: success,
			Error:   error,
		},
		LatencyMS: LatencyMS{
			P99: p99.Milliseconds(),
		},
	}
	json.NewEncoder(w).Encode(&out)
}

func (p *Proxy) Listen(port int) {
	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	t := stats{p.Stats}
	http.Handle("/stats", &t)

	tr := tracing(nextRequestID)
	lh := logging(p.log)
	x := tr(lh(p))
	http.Handle("/", x)
	//tracingLogger := tr(lh(http.HandlerFunc(p.proxy)))
	//server := http.Server{
	//Addr:    fmt.Sprintf(":%d", port),
	//Handler: tracingLogger,
	//}
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	// server.ListenAndServe()
}

type status int

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dst *httputil.ReverseProxy
	ts := time.Now()
	defer func() {
		te := time.Now().Sub(ts)
		p.Stats.Record(te)
	}()

	addr, service, err := p.resolve(r.URL)
	if err != nil {
		log.Fatal(err)
		return
	}

	if service != "" {
		if _, ok := p.routeMap[service]; ok {
			tmp := addr.Path
			addr.Path = ""
			dst = p.open(addr)
			r.URL = addr
			addr.Path = tmp
			dst.ModifyResponse = func(resp *http.Response) (err error) {
				if resp.StatusCode > 399 {
					p.Stats.IncError()
				}
				if resp.StatusCode >= 200 && resp.StatusCode <= 399 {
					p.Stats.IncSuccess()
				}
				return nil
			}

			dst.ServeHTTP(w, r)
			return
		}
	}

	http.Error(w, p.defaultBody, p.defaultStatus)
	p.Stats.IncError()
	return
}

func (p *Proxy) get(saddr string) *httputil.ReverseProxy {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pool, ok := p.conns[saddr]; ok {
		for conn := range pool {
			return conn
		}
	}

	return nil
}

func (p *Proxy) open(addr *url.URL) *httputil.ReverseProxy {
	saddr := addr.String()
	c := p.get(saddr)
	if c != nil {
		return c
	}

	upstream, _ := url.Parse(saddr)
	proxy := httputil.NewSingleHostReverseProxy(upstream)

	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.conns[saddr]; !ok {
		p.conns[saddr] = make(map[*httputil.ReverseProxy]struct{})
	}
	p.conns[saddr][proxy] = struct{}{}

	return proxy
}

func (p *Proxy) matcherFn(uri string) (string, string) {
	service, path := "", ""
	for s, pattern := range p.routeMap {
		if ok := strings.Contains(uri, pattern); ok {
			service = s
			path = strings.TrimPrefix(uri, pattern)
			break
		}
	}
	return service, path
}

func (p *Proxy) resolve(upstream *url.URL) (*url.URL, string, error) {
	service, path := p.matcherFn(upstream.String())
	record := p.Scheduler.NextBackend(service)

	target := fmt.Sprintf("%s:%d", record.Target, record.Port)
	out := &url.URL{Scheme: "http", Host: target, Path: path}

	return out, service, nil
}
