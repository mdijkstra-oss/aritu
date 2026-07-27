package scenario

import "strings"

type Param struct {
	Name  string
	Value string
}

type HeaderValue struct {
	Token  string
	Params []Param
}

func ParseHeaderValue(raw string) HeaderValue {
	segments := strings.Split(raw, ";")
	token := strings.TrimSpace(segments[0])
	if token == "" {
		return HeaderValue{}
	}

	params := make([]Param, 0, len(segments)-1)
	for _, segment := range segments[1:] {
		params = append(params, parseParam(segment))
	}
	return HeaderValue{Token: token, Params: params}
}

func parseParam(segment string) Param {
	name, value, hasValue := strings.Cut(strings.TrimSpace(segment), "=")
	if !hasValue {
		return Param{Name: strings.TrimSpace(name)}
	}
	return Param{Name: strings.TrimSpace(name), Value: unquote(strings.TrimSpace(value))}
}

func unquote(value string) string {
	if !strings.HasPrefix(value, `"`) {
		return value
	}
	inner := value[1:]
	if strings.HasSuffix(inner, `"`) {
		return inner[:len(inner)-1]
	}
	return inner
}
