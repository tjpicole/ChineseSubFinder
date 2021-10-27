package vad

import (
	"bufio"
	"errors"
	"fmt"
	webRTCVAD "github.com/baabaaox/go-webrtcvad"
	"io"
	"os"
	"time"
)

// GetVADInfo 分析音频文件，得到 VAD 分析信息，看样子是不支持并发的，只能单线程使用
func GetVADInfo(audioInfo AudioInfo) ([]VADInfo, error) {

	var (
		frameIndex  = 0
		frameSize   = audioInfo.SampleRate / 1000 * FrameDuration
		frameBuffer = make([]byte, audioInfo.SampleRate/1000*FrameDuration*audioInfo.BitDepth/8)
		frameActive = false
		vadInfos    = make([]VADInfo, 0)
	)

	audioFile, err := os.Open(audioInfo.FileFullPath)
	if err != nil {
		return nil, err
	}
	defer audioFile.Close()

	reader := bufio.NewReader(audioFile)

	vadInst := webRTCVAD.Create()
	defer webRTCVAD.Free(vadInst)

	err = webRTCVAD.Init(vadInst)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	err = webRTCVAD.SetMode(vadInst, Mode)
	if err != nil {
		return nil, err
	}

	if ok := webRTCVAD.ValidRateAndFrameLength(audioInfo.SampleRate, frameSize); !ok {
		return nil, errors.New(fmt.Sprintf("invalid rate or frame length, %v", audioInfo.FileFullPath))
	}
	var offset int

	report := func() {
		t := time.Duration(offset) * time.Second / time.Duration(audioInfo.SampleRate) / 2
		//log.Printf("Frame: %v, offset: %v, Active: %v, t = %v", frameIndex, offset, frameActive, t)
		vadInfos = append(vadInfos, VADInfo{
			Active: frameActive,
			Time:   t,
		})
	}

	for {
		_, err = io.ReadFull(reader, frameBuffer)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		tmpFrameActive, err := webRTCVAD.Process(vadInst, audioInfo.SampleRate, frameBuffer, frameSize)
		if err != nil {
			return nil, err
		}
		if tmpFrameActive != frameActive || offset == 0 {
			frameActive = tmpFrameActive
			report()
		}
		offset += len(frameBuffer)
		frameIndex++
	}

	report()

	return vadInfos, nil
}

type VADInfo struct {
	Frame  int           // 第几帧
	Offset int           // 音频的偏移
	Active bool          // 当前帧（时间窗口）是否检测到语音
	Time   time.Duration // 时间点
}

const (
	// Mode vad mode，VAD 的模式
	Mode = 2
	// FrameDuration frame duration，分析的时间窗口
	FrameDuration = 10
)
