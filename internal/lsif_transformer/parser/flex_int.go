package parser

import (
	"encoding/json"
	"strconv"
)

type FlexInt uint32

func (fi *FlexInt) UnmarshalJSON(b []byte) error {
	if b[0] != '"' {
		return json.Unmarshal(b, (*uint32)(fi))
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	*fi = FlexInt(i)

	return nil
}
