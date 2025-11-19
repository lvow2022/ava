package audio

type Frame struct {
	Payload []byte
	IsFirst bool
	IsLast  bool
}

type CodecOption struct {
	Codec         string `json:"codec" default:"pcm"`
	SampleRate    int    `json:"sampleRate" default:"16000"`
	Channels      int    `json:"channels" default:"1"`
	BitDepth      int    `json:"bitDepth" default:"16"`
	FrameDuration string `json:"frameDuration" default:"20ms"`
	PayloadType   uint8  `json:"payloadType" default:"1"`
}
