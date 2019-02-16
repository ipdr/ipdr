package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	files "github.com/ipfs/go-ipfs-cmdkit/files"
)

func (s *Shell) DagGet(ref string, out interface{}) error {
	return s.Request("dag/get", ref).Exec(context.Background(), out)
}

func (s *Shell) DagPut(data interface{}, ienc, kind string) (string, error) {
	var r io.Reader
	switch data := data.(type) {
	case string:
		r = strings.NewReader(data)
	case []byte:
		r = bytes.NewReader(data)
	case io.Reader:
		r = data
	default:
		return "", fmt.Errorf("cannot current handle putting values of type %T", data)
	}

	rc := ioutil.NopCloser(r)
	fr := files.NewReaderFile("", "", rc, nil)
	slf := files.NewSliceFile("", "", []files.File{fr})
	fileReader := files.NewMultiFileReader(slf, true)

	var out struct {
		Cid struct {
			Target string `json:"/"`
		}
	}

	return out.Cid.Target, s.
		Request("dag/put").
		Option("input-enc", ienc).
		Option("format", kind).
		Body(fileReader).
		Exec(context.Background(), &out)
}
