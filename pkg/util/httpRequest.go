package util

import (
	"fmt"
	"sort"
)

func ConvertMapParamsToString(params map[string]interface{}) string {
	keys := make([]string, len(params))
	i := 0
	_val := ""
	for k, _ := range params {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		_val += k + "=" + fmt.Sprintf("%v", params[k]) + "&"
	}
	_val = _val[0 : len(_val)-1]

	return _val
}
