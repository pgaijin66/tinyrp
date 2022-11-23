package server

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

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
	mux := http.NewServeMux()

	for _, resource := range config.Resources {
		url, _ := url.Parse(resource.Destination_URL)
		proxy := NewProxy(url)
		mux.HandleFunc(resource.Endpoint, ProxyRequestHandler(proxy, url, resource.Endpoint))
	}

	http.ListenAndServe(":8080", mux)
	return nil

}

func NewProxy(target *url.URL) *httputil.ReverseProxy {
	// We will get to this...
	// url, _ := url.Parse(target)
	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)
	return proxy
}

func ProxyRequestHandler(proxy *httputil.ReverseProxy, url *url.URL, endpoint string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[ TinyRP ] Request received at %s at %s\n", r.URL, time.Now().UTC())
		// Update the headers to allow for SSL redirection
		r.URL.Host = url.Host
		r.URL.Scheme = url.Scheme
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Host = url.Host

		//trim reverseProxyRoutePrefix
		path := r.URL.Path
		r.URL.Path = strings.TrimLeft(path, endpoint)

		// Note that ServeHttp is non blocking and uses a go routine under the hood
		fmt.Printf("[ TinyRP ] Redirecting request to %s at %s\n", r.URL, time.Now().UTC())
		proxy.ServeHTTP(w, r)

	}
}
