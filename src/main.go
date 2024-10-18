package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type unitServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

type Server interface {
	Address() string
	IsAlive() bool
	Serve(w http.ResponseWriter, r *http.Request)
}

type Loadbalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

func main() {
	servers := []Server{
		newUnitServer("https://jsonplaceholder.typicode.com/posts/1"),
		newUnitServer("https://jsonplaceholder.typicode.com/posts/8"),
		newUnitServer("https://jsonplaceholder.typicode.com/posts/7"),
	}

	lb := newLoadbalancer("8080", servers)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s] %s%s\n", r.Method, r.Host, r.URL.Path)
		lb.serveProxy(w, r)
	})

	http.Handle("/", handler)
	fmt.Printf("serving at localhost:%s\n", lb.port)
	if err := http.ListenAndServe(":"+lb.port, nil); err != nil {
		fmt.Printf("server error: %v\n", err)
		os.Exit(1)
	}
}

func (s *unitServer) Address() string {
	return s.addr
}

func (s *unitServer) IsAlive() bool {
	return true
}

func (s *unitServer) Serve(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

func (lb *Loadbalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	lb.roundRobinCount++
	fmt.Printf("Selected server: %s\n", server.Address())
	return server
}

func (lb *Loadbalancer) serveProxy(w http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvailableServer()
	targetServer.Serve(w, r)
}

func newLoadbalancer(port string, servers []Server) *Loadbalancer {
	return &Loadbalancer{
		port:            port,
		servers:         servers,
		roundRobinCount: 0,
	}
}

func newUnitServer(addr string) *unitServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	proxy := httputil.NewSingleHostReverseProxy(serverUrl)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = serverUrl.Host
		req.URL.Scheme = serverUrl.Scheme
		req.URL.Host = serverUrl.Host
	}

	return &unitServer{
		addr:  addr,
		proxy: proxy,
	}
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
