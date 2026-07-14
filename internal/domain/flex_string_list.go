package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FlexStringList 容错 JSON 字符串列表：接受 []string、单个 string、或 object（展平为短语）。
type FlexStringList []string

func (l *FlexStringList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*l = nil
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*l = FlexStringList(normalizeStringList(arr))
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*l = FlexStringList(normalizeStringList([]string{s}))
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err == nil {
		out := make([]string, 0, len(obj))
		for k, v := range obj {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if item := flexStringListFromRaw(v); item != "" {
				out = append(out, k+": "+item)
			}
		}
		*l = FlexStringList(normalizeStringList(out))
		return nil
	}
	return fmt.Errorf("flex string list: unsupported JSON %s", string(data))
}

func flexStringListFromRaw(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return strings.Join(normalizeStringList(list), "；")
	}
	return ""
}

func normalizeStringList(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (l FlexStringList) strings() []string {
	if len(l) == 0 {
		return nil
	}
	return []string(l)
}
