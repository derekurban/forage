package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/derekurban/forage/internal/apperr"
)

func TestWriteErrorJSONEnvelope(t *testing.T) {
	var buf bytes.Buffer
	err := WriteError(&buf, Options{JSON: true}, "forage test", apperr.New(apperr.CodeNotImplemented, "not ready", apperr.ExitNoProvider))
	if err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.OK {
		t.Fatal("OK = true")
	}
	if env.Error == nil || env.Error.Code != apperr.CodeNotImplemented {
		t.Fatalf("error = %+v", env.Error)
	}
}
