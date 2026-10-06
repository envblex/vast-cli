package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	OffersAlias = map[string]string{
		"cuda_vers":      "cuda_max_good",
		"display_active": "gpu_display_active",
		"dlperf_usd":     "dlperf_per_dphtotal",
		"dph":            "dph_total",
		"flops_usd":      "flops_per_dphtotal",
	}

	OffersMultiplier = map[string]float64{
		"cpu_ram":       1000.0,
		"gpu_ram":       1000.0,
		"gpu_total_ram": 1000.0,
		"duration":      86400.0,
	}

	OpNames = map[string]string{
		">=":      "gte",
		">":       "gt",
		"gt":      "gt",
		"gte":     "gte",
		"<=":      "lte",
		"<":       "lt",
		"lt":      "lt",
		"lte":     "lte",
		"!=":      "neq",
		"==":      "eq",
		"=":       "eq",
		"eq":      "eq",
		"neq":     "neq",
		"noteq":   "neq",
		"not eq":  "neq",
		"notin":   "notin",
		"not in":  "notin",
		"nin":     "notin",
		"in":      "in",
	}

	// Regex matching field op value
	// Example: gpu_name=RTX_4090 or num_gpus>=1 or dph<=2.0
	termRegex = regexp.MustCompile(`([a-zA-Z0-9_]+)\s*([=><!]+|\b(?:gte|gt|lte|lt|neq|eq|notin|nin|in|not\s+eq|not\s+in)\b)\s*(\[[^\]]+\]|"[^"]*"|'[^']*'|[^\s]+)`)
)

// ParseOrder parses an order string like "dlperf_usd-,dph" into [["dlperf_per_dphtotal", "desc"], ...]
func ParseOrder(orderStr string) [][]string {
	if strings.TrimSpace(orderStr) == "" {
		return [][]string{{"score", "desc"}}
	}

	parts := strings.Split(orderStr, ",")
	res := make([][]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		dir := "asc"
		field := p
		if strings.HasSuffix(field, "-") {
			dir = "desc"
			field = strings.TrimSuffix(field, "-")
		} else if strings.HasPrefix(field, "-") {
			dir = "desc"
			field = strings.TrimPrefix(field, "-")
		} else if strings.HasSuffix(field, "+") {
			dir = "asc"
			field = strings.TrimSuffix(field, "+")
		} else if strings.HasPrefix(field, "+") {
			dir = "asc"
			field = strings.TrimPrefix(field, "+")
		}

		if alias, ok := OffersAlias[field]; ok {
			field = alias
		}
		res = append(res, []string{field, dir})
	}

	if len(res) == 0 {
		return [][]string{{"score", "desc"}}
	}
	return res
}

func parseValue(valStr string) interface{} {
	valStr = strings.TrimSpace(valStr)
	// Quoted string
	if (strings.HasPrefix(valStr, `"`) && strings.HasSuffix(valStr, `"`)) ||
		(strings.HasPrefix(valStr, `'`) && strings.HasSuffix(valStr, `'`)) {
		return valStr[1 : len(valStr)-1]
	}

	// Array
	if strings.HasPrefix(valStr, "[") && strings.HasSuffix(valStr, "]") {
		inner := valStr[1 : len(valStr)-1]
		parts := strings.Split(inner, ",")
		var arr []interface{}
		for _, p := range parts {
			arr = append(arr, parseValue(p))
		}
		return arr
	}

	// Boolean
	lower := strings.ToLower(valStr)
	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}

	// Integer
	if intVal, err := strconv.ParseInt(valStr, 10, 64); err == nil {
		return intVal
	}

	// Float
	if floatVal, err := strconv.ParseFloat(valStr, 64); err == nil {
		return floatVal
	}

	return valStr
}

// ParseQuery parses a Vast.ai filter query string into a query map
// e.g. "gpu_name=RTX_4090 num_gpus>=1 verified=true" ->
// { "gpu_name": {"eq": "RTX_4090"}, "num_gpus": {"gte": 1}, "verified": {"eq": true} }
func ParseQuery(queryStr string) (map[string]map[string]interface{}, error) {
	queryStr = strings.TrimSpace(queryStr)
	res := make(map[string]map[string]interface{})
	if queryStr == "" {
		return res, nil
	}

	matches := termRegex.FindAllStringSubmatch(queryStr, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("could not parse query filter: %s", queryStr)
	}

	for _, m := range matches {
		field := m[1]
		opRaw := strings.ToLower(strings.TrimSpace(m[2]))
		valRaw := m[3]

		// Resolve alias
		if alias, ok := OffersAlias[field]; ok {
			field = alias
		}

		op, ok := OpNames[opRaw]
		if !ok {
			return nil, fmt.Errorf("unsupported operator: %s", opRaw)
		}

		val := parseValue(valRaw)
		if strVal, ok := val.(string); ok {
			val = strings.ReplaceAll(strVal, "_", " ")
		} else if arrVal, ok := val.([]interface{}); ok {
			var newArr []interface{}
			for _, item := range arrVal {
				if s, ok := item.(string); ok {
					newArr = append(newArr, strings.ReplaceAll(s, "_", " "))
				} else {
					newArr = append(newArr, item)
				}
			}
			val = newArr
		}

		// Apply multiplier if applicable and value is numeric
		if mult, ok := OffersMultiplier[field]; ok {
			switch v := val.(type) {
			case int64:
				val = float64(v) * mult
			case float64:
				val = v * mult
			}
		}

		if _, exists := res[field]; !exists {
			res[field] = make(map[string]interface{})
		}
		res[field][op] = val
	}

	return res, nil
}
