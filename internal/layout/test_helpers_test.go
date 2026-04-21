package layout

import "encoding/json"

type testMapCodec struct{}

func (testMapCodec) Marshal(value map[string]string) ([]byte, error) {
	return json.Marshal(value)
}

func (testMapCodec) Unmarshal(data []byte) (map[string]string, error) {
	var value map[string]string
	err := json.Unmarshal(data, &value)
	return value, err
}

type testMapFile struct {
	Format[map[string]string, testMapCodec]
}
