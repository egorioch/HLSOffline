package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/deepch/vdk/av"
)

var (
	Success                         = "success"
	ErrorStreamNotFound             = errors.New("stream not found")
	ErrorStreamAlreadyExists        = errors.New("stream already exists")
	ErrorStreamChannelAlreadyExists = errors.New("stream channel already exists")
	ErrorStreamNotHLSSegments       = errors.New("stream hls not ts seq found")
	ErrorStreamNoVideo              = errors.New("stream no video")
	ErrorStreamNoClients            = errors.New("stream no clients")
	ErrorStreamRestart              = errors.New("stream restart")
	ErrorStreamStopCoreSignal       = errors.New("stream stop core signal")
	ErrorStreamStopRTSPSignal       = errors.New("stream stop rtsp signal")
	ErrorStreamChannelNotFound      = errors.New("stream channel not found")
	ErrorStreamChannelCodecNotFound = errors.New("stream channel codec not ready, possible stream offline")
	ErrorStreamsLen0                = errors.New("streams len zero")
)

// Config global
var Config = loadConfig()

// ConfigST struct
type ConfigST struct {
	mutex   sync.RWMutex
	Server  ServerST            `json:"server"`
	Streams map[string]StreamST `json:"streams"`
}

// ServerST struct
type ServerST struct {
	HTTPPort string `json:"http_port"`
}

// StreamST struct
type StreamST struct {
	URL              string          `json:"url"`
	Status           bool            `json:"status"`
	OnDemand         bool            `json:"on_demand"`
	RunLock          bool            `json:"-"`
	hlsSegmentNumber int             `json:"-"`
	hlsSegmentBuffer map[int]Segment `json:"-"`
	Codecs           []av.CodecData
	Cl               map[string]viewer
}

// Segment HLS cache section
type Segment struct {
	dur  time.Duration
	data []*av.Packet
}

type viewer struct {
	c chan av.Packet
}

func readSegments(segments map[int]Segment) {
	log.Println("[StreamHLSm3u8] -> config[readSegments]:")
	for i, val := range segments {
		log.Printf("[%d]: segmentDuration %s\n", i, val.dur)
		log.Printf("[pck := range val.data](idx av.data)%d: %s", 0, printAVPacketsData(*val.data[0]))
		log.Printf("[pck := range val.data](idx av.data)%d: %s", len(val.data)-1/2, printAVPacketsData(*val.data[(len(val.data)-1)/2]))
		log.Printf("[pck := range val.data](idx av.data)%d: %s", len(val.data)-1, printAVPacketsData(*val.data[len(val.data)-1]))

	}
}

func printAVPacketsData(packet av.Packet) string {
	return fmt.Sprintf(
		"config[printAVPacketsData]: IsKeyFrame: %t, Duration: %s, Idx: %d, CompositionTime %s, Time: %s \n",
		packet.IsKeyFrame, packet.Duration, packet.Idx, packet.CompositionTime, packet.Time,
	)
}

func (element *ConfigST) RunIFNotRun(uuid string) {
	element.mutex.Lock()
	log.Println("[main.http -> main.config[RunIFNotRun]]")
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		if tmp.OnDemand && !tmp.RunLock {
			tmp.RunLock = true
			element.Streams[uuid] = tmp
			go RTSPWorkerLoop(uuid, tmp.URL, tmp.OnDemand)
		}
	}
}

func (element *ConfigST) RunUnlock(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		if tmp.OnDemand && tmp.RunLock {
			tmp.RunLock = false
			element.Streams[uuid] = tmp
		}
	}
}

func (element *ConfigST) HasViewer(uuid string) bool {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok && len(tmp.Cl) > 0 {
		return true
	}
	return false
}

func loadConfig() *ConfigST {
	var tmp ConfigST
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		log.Fatalln(err)
	}
	for i, v := range tmp.Streams {
		v.Cl = make(map[string]viewer)
		v.hlsSegmentBuffer = make(map[int]Segment)
		tmp.Streams[i] = v
	}
	return &tmp
}

func (element *ConfigST) cast(uuid string, pck av.Packet) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	for _, v := range element.Streams[uuid].Cl {
		if len(v.c) < cap(v.c) {
			v.c <- pck
		}
	}
}

func (element *ConfigST) ext(suuid string) bool {
	element.mutex.Lock()
	log.Printf("[main.config] element(HTTPPort): %s \n", element.Server.HTTPPort)
	defer element.mutex.Unlock()
	_, ok := element.Streams[suuid]
	return ok
}

func (element *ConfigST) coAd(suuid string, codecs []av.CodecData) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	t := element.Streams[suuid]
	t.Codecs = codecs
	element.Streams[suuid] = t
}

func (element *ConfigST) coGe(suuid string) []av.CodecData {
	for i := 0; i < 100; i++ {
		element.mutex.RLock()
		tmp, ok := element.Streams[suuid]
		element.mutex.RUnlock()
		if !ok {
			return nil
		}
		if tmp.Codecs != nil {
			return tmp.Codecs
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func (element *ConfigST) clAd(suuid string) (string, chan av.Packet) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	cuuid := pseudoUUID()
	ch := make(chan av.Packet, 100)
	element.Streams[suuid].Cl[cuuid] = viewer{c: ch}
	return cuuid, ch
}

func (element *ConfigST) list() (string, []string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	var res []string
	var fist string
	for k := range element.Streams {
		if fist == "" {
			fist = k
		}
		res = append(res, k)
	}
	return fist, res
}
func (element *ConfigST) clDe(suuid, cuuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	delete(element.Streams[suuid].Cl, cuuid)
}

func pseudoUUID() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	log.Printf("config[pseudoUUID]: %s \n", uuid)
	return
}

// StreamHLSAdd add hls seq to buffer
func (obj *ConfigST) StreamHLSAdd(uuid string, val []*av.Packet, dur time.Duration) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if tmp, ok := obj.Streams[uuid]; ok {
		tmp.hlsSegmentNumber++
		tmp.hlsSegmentBuffer[tmp.hlsSegmentNumber] = Segment{data: val, dur: dur}
		if len(tmp.hlsSegmentBuffer) >= 6 {
			delete(tmp.hlsSegmentBuffer, tmp.hlsSegmentNumber-6-1)
		}
		obj.Streams[uuid] = tmp
	}
}

// StreamHLSm3u8 get hls m3u8 list
func (obj *ConfigST) StreamHLSm3u8(uuid string) (string, int, error) {
	log.Println("[config[StreamHLSm3u8]] Получение hls-list m3u8")
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	if tmp, ok := obj.Streams[uuid]; ok {
		var out string
		//TODO fix  it
		out += "#EXTM3U\r\n#EXT-X-TARGETDURATION:4\r\n#EXT-X-VERSION:4\r\n#EXT-X-MEDIA-SEQUENCE:" + strconv.Itoa(tmp.hlsSegmentNumber) + "\r\n"
		var keys []int
		// segmentBuffer --набор байтов
		for k := range tmp.hlsSegmentBuffer {
			keys = append(keys, k)
		}
		log.Println("config[StreamHLSm3u8]: hls segment buffer")
		readSegments(tmp.hlsSegmentBuffer)

		sort.Ints(keys)
		var count int
		for _, i := range keys {
			count++
			out += "#EXTINF:" + strconv.FormatFloat(tmp.hlsSegmentBuffer[i].dur.Seconds(), 'f', 1, 64) + ",\r\nsegment/" + strconv.Itoa(i) + "/file.ts\r\n"

		}
		return out, count, nil
	}
	return "", 0, ErrorStreamNotFound
}

// StreamHLSTS send hls segment buffer to clients
func (obj *ConfigST) StreamHLSTS(uuid string, seq int) ([]*av.Packet, error) {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	if tmp, ok := obj.Streams[uuid]; ok {
		if buf, ok := tmp.hlsSegmentBuffer[seq]; ok {
			return buf.data, nil
		}
	}
	return nil, ErrorStreamNotFound
}

// StreamHLSFlush delete hls cache
func (obj *ConfigST) StreamHLSFlush(uuid string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if tmp, ok := obj.Streams[uuid]; ok {
		tmp.hlsSegmentBuffer = make(map[int]Segment)
		tmp.hlsSegmentNumber = 0
		obj.Streams[uuid] = tmp
	}
}

// stringToInt convert string to int if err to zero
func stringToInt(val string) int {
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return i
}
