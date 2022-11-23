package server

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/pgaijin66/lightweight-reverse-proxy/internal/configs"
)

// ping returns a "pong" message
func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

// type Prox struct {
// 	target *url.URL
// 	proxy  *httputil.ReverseProxy
// }

// func NewProxy(target string) *Prox {
// 	url, _ := url.Parse(target)
// 	return &Prox{
// 		target: url,
// 		proxy:  httputil.NewSingleHostReverseProxy(url)}

// }

// func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		fmt.Println("here")
// 		r.Host = "localhost:9001"
// 		proxy.ServeHTTP(w, r)
// 	}
// }

// Run starts server and listens on defined port
func Run() error {
	// load configurations from config file
	config, err := configs.NewConfiguration()
	if err != nil {
		log.Fatalf("could not load configuration: %v", err)
	}

	fmt.Println(config.Resources)

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRequestAndRedirect1)
	mux.HandleFunc("/server2", handleRequestAndRedirect2)
	mux.HandleFunc("/server3", handleRequestAndRedirect3)

	http.ListenAndServe(":8080", mux)

	return nil

}

// Given a request send it to the appropriate url
func handleRequestAndRedirect1(res http.ResponseWriter, req *http.Request) {
	// We will get to this...
	url, _ := url.Parse("http://localhost:9001")
	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect2(res http.ResponseWriter, req *http.Request) {
	// We will get to this...
	url, _ := url.Parse("http://localhost:9002")
	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	//trim reverseProxyRoutePrefix
	path := req.URL.Path
	req.URL.Path = strings.TrimLeft(path, "/server2")

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect3(res http.ResponseWriter, req *http.Request) {
	// We will get to this...
	url, _ := url.Parse("http://localhost:9003")
	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	//trim reverseProxyRoutePrefix
	path := req.URL.Path
	req.URL.Path = strings.TrimLeft(path, "/server3")

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}
