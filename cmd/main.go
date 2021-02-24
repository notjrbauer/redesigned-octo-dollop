package main

import (
	"flag"
	"log"
	"os"
	"time"

	gateway "github.com/notjrbauer/interview/rippling"
)

func main() {
	var cfgPath string
	var dsocketPath string
	var port int

	flag.StringVar(&cfgPath, "config path", "config.yml", "configuration for api-gateway")
	flag.StringVar(&dsocketPath, "docker socket abs path", "/var/run/docker.sock", "docker sock abs path")
	flag.IntVar(&port, "port", 8080, "Port to serve api-gateway")
	flag.Parse()

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("The Gateway ooooo")
	logger.Println("Serving on:", port)

	cfg, err := gateway.ParseConfig(cfgPath)
	check(err)

	// Load routes && tags for mapping
	routes := cfg.Routes
	tags := map[string][]string{}
	for _, t := range cfg.Backends {
		tags[t.Name] = t.MatchLabels
	}

	s := gateway.NewScheduler(time.Second, nil, tags, dsocketPath)
	p := gateway.NewProxy(routes, logger)

	if resp := cfg.DefaultResponse; resp.Body != "" && resp.StatusCode != 0 {
		p.SetDefaultResponse(resp.StatusCode, resp.Body)
	}
	p.Scheduler = s
	p.Listen(3000)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
