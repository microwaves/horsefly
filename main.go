package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"

	"github.com/microwaves/go-utils/logger"
	"rsc.io/letsencrypt"
)

var (
	confFile             = flag.String("conf", "", "Configuration file")
	httpAddr             = flag.String("http", ":http", "HTTP listen address")
	letsEncryptCacheFile = flag.String("letsencrypt-cache", "", "Let's Encrypt cache file (default HTTPS disabled)")
	logFile              = flag.String("log", "/var/log/horsefly.log", "Log file")
	pollInterval         = flag.Duration("poll", time.Second*10, "Configuration file poll interval")
)

type Conf struct {
	Host    string
	Forward string
	Serve   string

	handler http.Handler
}

func makeHandler(c *Conf) http.Handler {
	if h := c.Forward; h != "" {
		return &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = h
			},
		}
	}

	if d := c.Serve; d != "" {
		return http.FileServer(http.Dir(d))
	}

	return nil
}

func parseConf(file string) ([]*Conf, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var conf []*Conf
	if err := json.NewDecoder(f).Decode(&conf); err != nil {
		return nil, err
	}

	for _, c := range conf {
		c.handler = makeHandler(c)
		if c.handler == nil {
			logger.Error.Printf("bad configuration: %#v", c)
		}
	}

	return conf, nil
}

func listen(fwd int, addr string) (l net.Listener) {
	var err error

	if fwd >= 3 {
		l, err = net.FileListener(os.NewFile(uintptr(fwd), "https"))
	} else {
		l, err = net.Listen("tcp", addr)
	}
	handleError(err)

	return
}

func main() {
	flag.Parse()
	setupLogging()

	s, err := NewServer(*confFile, *pollInterval)
	handleError(err)

	httpFWD, _ := strconv.Atoi(os.Getenv("RUNSIT_PORTFD_http"))
	httpsFWD, _ := strconv.Atoi(os.Getenv("RUNSIT_PORTFD_https"))

	if *letsEncryptCacheFile != "" {
		var m letsencrypt.Manager

		err := m.CacheFile(*letsEncryptCacheFile)
		handleError(err)

		cfg := tls.Config{GetCertificate: m.GetCertificate}
		l := tls.NewListener(listen(httpsFWD, ":https"), &cfg)

		go func() {
			err := http.Serve(l, s)
			handleError(err)
		}()
	}

	err = http.Serve(listen(httpFWD, *httpAddr), s)
	handleError(err)
}
