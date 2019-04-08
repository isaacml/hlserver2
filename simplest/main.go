package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

var rootdir = "/var/segments/"

func main() {
	cert := "/etc/letsencrypt/live/todostreaming.es/cert.pem"
	key := "/etc/letsencrypt/live/todostreaming.es/privkey.pem"

	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServeTLS(":443", cert, key, nil))
}

func root(w http.ResponseWriter, r *http.Request) {
	namefile := strings.TrimRight(rootdir+r.URL.Path[1:], "/")
	fileinfo, err := os.Stat(namefile)
	if err != nil {
		// fichero no existe
		http.NotFound(w, r)
		return
	}
	fr, errn := os.Open(namefile)
	if errn != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}
	defer fr.Close()

	if strings.Contains(namefile, ".m3u8") {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileinfo.Size()))
		w.Header().Set("Accept-Ranges", "bytes")
	} else if strings.Contains(namefile, ".ts") {
		w.Header().Set("Cache-Control", "max-age=300")
		w.Header().Set("Content-Type", "video/MP2T")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileinfo.Size()))
		w.Header().Set("Accept-Ranges", "bytes")
	}

	http.ServeContent(w, r, namefile, fileinfo.ModTime(), fr)
}
