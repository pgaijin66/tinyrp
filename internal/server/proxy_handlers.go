package server

// import (
// 	"net/http"
// 	"net/http/httputil"
// 	"net/url"
// )

// // NewProxy takes target host and creates a reverse proxy
// func NewProxy(target string) (*httputil.ReverseProxy, error) {
// 	url, _ := url.Parse(target)
// 	return httputil.NewSingleHostReverseProxy(url), nil
// }

// ProxyRequestHandler handles the http request using proxy
// func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		proxy.ServeHTTP(w, r)
// 	}
// }
