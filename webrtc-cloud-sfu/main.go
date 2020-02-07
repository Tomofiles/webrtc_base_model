package main

import (
	"flag"
	"fmt"
	"net/http"
	"regexp"

	"github.com/pion/webrtc/v2"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	m = webrtc.MediaEngine{}

	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	m.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	api = webrtc.NewAPI(webrtc.WithMediaEngine(m))
}

func main() {
	port := flag.String("p", "8080", "http port")
	serverURL = flag.String("s", "localhost:3478", "stun/turn server url")
	userCred := flag.String("u", "username=credential", "username & credential")
	flag.Parse()

	kv := regexp.MustCompile(`(\w+)=(\w+)`).FindStringSubmatch(*userCred)
	serverUname = kv[1]
	serverCredential = kv[2]

	http.HandleFunc("/ws", room)
	http.HandleFunc("/serverinfo", getServerInfo)
	http.HandleFunc("/", web)

	fmt.Println("Web listening :" + *port)
	panic(http.ListenAndServe(":"+*port, nil))
}
