package dashboard

import (
	"net/http"
//	"net/http/httputil"
//	"net/url"
)

func Start() {
	go StartGrpc()
	go StartRestGateway()
	go StaticFileServer()
}

func StaticFileServer() {
	http.Handle("/", http.FileServer(http.Dir("./webui")))
	http.ListenAndServe(":8080", nil)
}
