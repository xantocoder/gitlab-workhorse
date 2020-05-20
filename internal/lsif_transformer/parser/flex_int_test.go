package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type jsonWithFlexInt struct {
	Value FlexInt `json:"value"`
}

func TestFlexInt(t *testing.T) {
	var v jsonWithFlexInt
	require.NoError(t, json.Unmarshal([]byte(`{ "value": 123 }`), &v))
	require.Equal(t, FlexInt(123), v.Value)

	require.NoError(t, json.Unmarshal([]byte(`{ "value": "123" }`), &v))
	require.Equal(t, FlexInt(123), v.Value)
}
