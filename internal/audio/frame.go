package audio

type Frame struct {
	Payload []byte
	IsFirst bool
	IsLast  bool
}
