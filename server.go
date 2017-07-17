package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Server struct {
	mtx  sync.RWMutex
	last time.Time
	conf []*Conf
}

func (s *Server) loadConf(file string) error {
	f, err := os.Stat(file)
	if err != nil {
		return err
	}

	mt := f.ModTime()
	if !mt.After(s.last) && s.conf != nil {
		return nil
	}

	c, err := parseConf(file)
	if err != nil {
		return err
	}

	s.mtx.Lock()
	s.last = mt
	s.conf = c
	s.mtx.Unlock()

	return nil
}

func (s *Server) refreshConf(file string, poll time.Duration) {
	for {
		if err := s.loadConf(file); err != nil {
			log.Println(err)
		}

		time.Sleep(poll)
	}
}

func (s *Server) handler(req *http.Request) http.Handler {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	h := req.Host
	if i := strings.Index(h, ":"); i >= 0 {
		h = h[:i]
	}

	for _, c := range s.conf {
		if h == c.Host || strings.HasSuffix(h, "."+c.Host) {
			return c.handler
		}
	}

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h := s.handler(r); h != nil {
		h.ServeHTTP(w, r)
		return
	}

	http.Error(w, "Not found.", http.StatusNotFound)
}

func NewServer(cfgfile string, poll time.Duration) (*Server, error) {
	s := new(Server)
	if err := s.loadConf(cfgfile); err != nil {
		return nil, err
	}

	go s.refreshConf(cfgfile, poll)
	return s, nil
}
