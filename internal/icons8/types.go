package icons8

import (
	"bytes"
	"encoding/json"
)

// StringList decodes a field that Icons8 returns sometimes as a string and
// sometimes as an array of strings. The icon search and vector-similarity
// endpoints disagree on this for category, categoryApiCode and subcategory.
type StringList []string

func (s *StringList) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*s = nil
		return nil
	}
	if data[0] == '[' {
		var list []string
		if err := json.Unmarshal(data, &list); err != nil {
			return err
		}
		*s = list
		return nil
	}
	var one string
	if err := json.Unmarshal(data, &one); err != nil {
		return err
	}
	if one == "" {
		*s = nil
		return nil
	}
	*s = StringList{one}
	return nil
}

func (s StringList) First() string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
