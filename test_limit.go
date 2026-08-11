package main
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)
func main() {
	data := []byte(`{"foo":"bar"}`)
	r := bytes.NewReader(data)
	limitR := io.LimitReader(r, 10*1024*1024)
	var m map[string]string
	err := json.NewDecoder(limitR).Decode(&m)
	fmt.Printf("err: %v, m: %v\n", err, m)
}

