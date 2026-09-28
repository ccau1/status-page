package proto

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

// CodecName is the name of the JSON codec registered for gRPC communication.
const CodecName = "json"

type JSONCodec struct{}

func (JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (JSONCodec) Name() string {
	return CodecName
}

func init() {
	encoding.RegisterCodec(JSONCodec{})
}
