package parser

import (
	"encoding/json"
	"errors"
	"strconv"
)

type Id int32

func (id *Id) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] != '"' {
		if err := id.unmarshalInt(b); err != nil {
			return err
		}
	} else {
		if err := id.unmarshalString(b); err != nil {
			return err
		}
	}

	if *id < minId || *id > maxId {
		return errors.New("json: id is invalid")
	}

	return nil
}

func (id *Id) unmarshalString(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	*id = Id(i)

	return nil
}
