package proto

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	testEncoder struct {
	}

	// testStruct is a model which includes the supported write types: string, int, unsigned int and struct.
	testStruct struct {
		S string
		I int64
		U uint64
		D struct {
			S string
			I int64
			U uint64
		}
	}
)

func (t *testEncoder) WriteString(buf *bytes.Buffer, s string) error {
	buf.WriteString(s + "|")
	return nil
}

func (t *testEncoder) Write(buf *bytes.Buffer, v interface{}) error {
	buf.WriteString(fmt.Sprintf("%+v|", v))
	return nil
}

func Test_WireWrite(t *testing.T) {
	result := bytes.NewBuffer(nil)
	encoder := &testEncoder{}

	require.NoError(
		t,
		WireWrite(
			result,
			encoder,
			testStruct{
				S: "hello",
				I: -1234,
				U: 1234,
				D: struct {
					S string
					I int64
					U uint64
				}{
					S: "world",
					I: -4321,
					U: 4321,
				},
			},
		),
	)

	require.Equal(t, "hello|-1234|1234|world|-4321|4321|", result.String())
}
