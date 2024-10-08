package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"HLSOffline/package/format/ts"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ServeHTTP replaces gin-based router
func serveHTTP() {
	http.HandleFunc("/play/hls/", NetPlayHls)
	//http.HandleFunc("/play/hls/segment/", NetPlayHLSTS)
	//http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Wrap with CORS middleware
	log.Fatal(http.ListenAndServe("localhost:8083", corsMiddleware(http.DefaultServeMux)))
}

func NetPlayHls(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("URL: %s\n", r.URL.Path)

	if strings.Contains(r.URL.Path, "/segment/") {
		fmt.Printf("CONTAINS SEGMENT")
		NetPlayHLSTS(w, r)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	suuid := parts[3]
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	fmt.Printf("suuid: %s\n", suuid)
	if !Config.ext(suuid) {
		return
	}
	Config.RunIFNotRun(suuid)
	for i := 0; i < 40; i++ {
		index, seq, err := Config.StreamHLSm3u8(suuid)
		fmt.Printf("[Config.StreamHLSm3u8]index: %s, seq: %s \n", index, seq)
		if err != nil {
			log.Println(err)
			return
		}
		if seq >= 6 {
			_, err := w.Write([]byte(index))
			if err != nil {
				log.Println(err)
				return
			}
			return
		}
		log.Println("Play list not ready wait or try update page")
		time.Sleep(1 * time.Second)
	}
}

// PlayHLSTS send client ts segment
func NetPlayHLSTS(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	suuid := parts[3]
	seg := parts[5]
	fmt.Printf("[NetPlayHLSTS]suuid: %s; seg: %s\n", suuid, seg)

	if !Config.ext(suuid) {
		return
	}
	codecs := Config.coGe(suuid)
	if codecs == nil {
		return
	}
	outfile := bytes.NewBuffer([]byte{})
	Muxer := ts.NewMuxer(outfile)
	err := Muxer.WriteHeader(codecs)
	if err != nil {
		log.Println(err)
		return
	}
	Muxer.PaddingToMakeCounterCont = true
	seqData, err := Config.StreamHLSTS(suuid, stringToInt(seg))
	if err != nil {
		log.Println(err)
		return
	}
	if len(seqData) == 0 {
		log.Println(err)
		return
	}
	for _, v := range seqData {
		v.CompositionTime = 1
		err = Muxer.WritePacket(*v)
		if err != nil {
			log.Println(err)
			return
		}
	}
	err = Muxer.WriteTrailer()
	if err != nil {
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = w.Write(outfile.Bytes())

	if err != nil {
		log.Println(err)
		return
	}
}
