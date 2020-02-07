package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"regexp"
	"sync"
	"unsafe"

	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

var (
	signalingURL     *string
	stunturnURL      *string
	serverUname      string
	serverCredential string

	videoTrack *webrtc.Track
	audioTrack *webrtc.Track
)

var mu sync.Mutex

func main() {
	signalingURL = flag.String("w", "localhost:80", "signaling server url")
	stunturnURL = flag.String("s", "localhost:3478", "stun/turn server url")
	userCred := flag.String("u", "username=credential", "username & credential")
	flag.Parse()

	kv := regexp.MustCompile(`(\w+)=(\w+)`).FindStringSubmatch(*userCred)
	serverUname = kv[1]
	serverCredential = kv[2]

	ws, _, err := websocket.DefaultDialer.Dial("wss://"+*signalingURL+"/ws", nil)
	if err != nil {
		log.Fatal("websocket client connection error:", err)
	}
	defer ws.Close()

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:" + *stunturnURL},
			},
			{
				URLs:           []string{"turn:" + *stunturnURL},
				Credential:     serverCredential,
				CredentialType: webrtc.ICECredentialTypePassword,
				Username:       serverUname,
			},
		},
	})
	if err != nil {
		panic(err)
	}

	audioTrack, err = peerConnection.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "pion1")
	if err != nil {
		panic(err)
	}
	if _, err = peerConnection.AddTrack(audioTrack); err != nil {
		panic(err)
	}

	videoTrack, err = peerConnection.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "video", "pion2")
	if err != nil {
		panic(err)
	}
	if _, err = peerConnection.AddTrack(videoTrack); err != nil {
		panic(err)
	}

	peerConnection.OnICECandidate(func(iceCandidate *webrtc.ICECandidate) {
		if iceCandidate != nil {
			mu.Lock()
			defer mu.Unlock()
			jsonIC, _ := json.Marshal(iceCandidate)
			fmt.Println(string(jsonIC))
			err := ws.WriteMessage(websocket.TextMessage, jsonIC)
			if err != nil {
				panic(err)
			}
		}
	})

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	peerConnection.SetLocalDescription(offer)

	mu.Lock()
	fmt.Println(offer.Type)
	fmt.Println(offer.SDP)
	err = ws.WriteMessage(websocket.TextMessage, []byte(offer.SDP))
	mu.Unlock()
	if err != nil {
		panic(err)
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())

		if connectionState == webrtc.ICEConnectionStateConnected {
			go func() {
				conn, _ := net.ListenPacket("udp", "127.0.0.1:8889")
				defer conn.Close()

				buffer := make([]byte, 1500)
				for {
					n, _, _ := conn.ReadFrom(buffer)

					packet := rtp.Packet{}
					packet.Unmarshal(buffer[:n])
					packet.SSRC = audioTrack.SSRC()

					audioTrack.WriteRTP(&packet)
				}
			}()
			go func() {
				conn, _ := net.ListenPacket("udp", "127.0.0.1:8888")
				defer conn.Close()

				buffer := make([]byte, 1500)
				for {
					n, _, _ := conn.ReadFrom(buffer)

					packet := rtp.Packet{}
					packet.Unmarshal(buffer[:n])
					packet.SSRC = videoTrack.SSRC()

					videoTrack.WriteRTP(&packet)
				}
			}()
		}
	})

	go func() {
		mu.Lock()
		defer mu.Unlock()
		_, ansOffStr, err := ws.ReadMessage()
		if err != nil {
			panic(err)
		}

		sdpType := webrtc.SDPTypeAnswer
		sdpType.UnmarshalJSON(ansOffStr)

		ansOffSD := webrtc.SessionDescription{
			Type: sdpType,
			SDP:  *(*string)(unsafe.Pointer(&ansOffStr)),
		}

		fmt.Println(ansOffSD.Type)
		fmt.Println(ansOffSD.SDP)

		if err = peerConnection.SetRemoteDescription(ansOffSD); err != nil {
			panic(err)
		}
	}()

	select {}
}
