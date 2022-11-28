package adaptive

import (
	"encoding/json"
	"time"
	"unsafe"
)

// #cgo LDFLAGS: ${SRCDIR}/../build/libshared.a
// #include "ads.h"
import "C"


type AdaptiveContext struct {
	In  chan string
	ctx unsafe.Pointer

	last struct {
		audio *AudioMetric
		video *VideoMetrics
		network *NetworkMetric
	}

	GroupofPicture float64
}

//export handle_bitrate_change
func handle_bitrate_change(bitrate C.int) {
	
}

func NewAdsContext(InChan chan string) *AdaptiveContext {
	ret := &AdaptiveContext{
		last: struct{audio *AudioMetric; video *VideoMetrics; network *NetworkMetric}{
			audio: nil,
			video: nil,
			network: nil,
		},
	}
	ret.ctx = C.new_ads_context()

	ret.In = InChan
	go func() {
		for {
			metricRaw := <-ret.In
			var out map[string]interface{}
			json.Unmarshal([]byte(metricRaw), &out)



			switch out["type"] {
			case "video":
				video := VideoMetrics{}
				json.Unmarshal([]byte(metricRaw), &video)
				ret.handleVideoMetric(&video);
			case "audio":
				audio := AudioMetric{}
				json.Unmarshal([]byte(metricRaw), &audio)
				ret.handleAudioMetric(&audio);
			case "network":
				network := NetworkMetric{}
				json.Unmarshal([]byte(metricRaw), &network)
				ret.handleNetworkMetric(&network);
			}
		}
	}()

	return ret
}




func (ads *AdaptiveContext)handleVideoMetric(metric *VideoMetrics) {
	lastVideoMetric := ads.last.video;
	if lastVideoMetric == nil {
		ads.last.video = metric
		return
	}

	timedif := time.Duration(metric.Timestamp - lastVideoMetric.Timestamp); //nanosecond

	decodedFps 					:= ( metric.FramesDecoded - 	lastVideoMetric.FramesDecoded ) 		/ (timedif.Seconds());
	receivedFps 				:= ( metric.FramesReceived - 	lastVideoMetric.FramesReceived) 		/ (timedif.Seconds());
	videoBandwidthConsumption 	:= ( metric.BytesReceived - 	lastVideoMetric.BytesReceived) 			/ (timedif.Seconds());
	decodeTimePerFrame 			:= ( metric.TotalDecodeTime - 	lastVideoMetric.TotalDecodeTime) 		/ (metric.FramesDecoded - lastVideoMetric.FramesDecoded)
	videoPacketsLostpercent     := ( metric.PacketsLost - 		lastVideoMetric.PacketsLost) 			/ (metric.PacketsReceived - lastVideoMetric.PacketsReceived);	
	videoJitter					:= metric.Jitter
	videoJitterBufferDelay		:= metric.JitterBufferDelay

	{
		ads.GroupofPicture =   metric.FramesDecoded - metric.KeyFramesDecoded;
	}

	C.ads_push_frame_decoded_per_second               (ads.ctx, C.int(decodedFps));
	C.ads_push_frame_received_per_second              (ads.ctx, C.int(receivedFps));
	C.ads_push_video_incoming_bandwidth_consumption   (ads.ctx, C.int(videoBandwidthConsumption));
	C.ads_push_decode_time_per_frame                  (ads.ctx, C.int(decodeTimePerFrame));
	C.ads_push_video_packets_lost                     (ads.ctx, C.float(videoPacketsLostpercent));
	C.ads_push_video_jitter                           (ads.ctx, C.int(videoJitter));
	C.ads_push_video_jitter_buffer_delay              (ads.ctx, C.int(videoJitterBufferDelay));

	ads.last.video = metric
}

func (ads *AdaptiveContext)handleNetworkMetric(metric *NetworkMetric) {
	lastNetworkMetric := ads.last.network;
	if lastNetworkMetric == nil {
		ads.last.network = metric
		return
	}

	timedif := time.Duration(metric.Timestamp - lastNetworkMetric.Timestamp); //nanosecond

	totalBandwidthConsumption  := (metric.BytesReceived - lastNetworkMetric.BytesReceived) / (timedif.Seconds());
	RTT 					   := metric.CurrentRoundTripTime * float64(time.Second.Nanoseconds());
	availableIncomingBandwidth := metric.AvailableIncomingBitrate 

	C.ads_push_rtt                                    (ads.ctx, C.int(RTT));
	C.ads_push_total_incoming_bandwidth_consumption   (ads.ctx, C.int(totalBandwidthConsumption));
	C.ads_push_available_incoming_bandwidth           (ads.ctx, C.int(availableIncomingBandwidth));

	ads.last.network = lastNetworkMetric
}

func (ads *AdaptiveContext)handleAudioMetric(metric *AudioMetric) {
	lastAudioMetric := ads.last.audio;
	if lastAudioMetric == nil {
		ads.last.audio = metric
		return
	}
	timedif := time.Duration(metric.Timestamp - lastAudioMetric.Timestamp); //nanosecond


	audioBandwidthConsumption 	:= ( metric.BytesReceived - 	lastAudioMetric.BytesReceived) 			/ (timedif.Seconds());
	C.ads_push_audio_incoming_bandwidth_consumption(ads.ctx,C.int(audioBandwidthConsumption));

	ads.last.audio = lastAudioMetric
}
