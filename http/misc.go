package httpapi

import (
	"sort"
	"strconv"
)

func (es Episodes) Len() int {
	return len(es)
}

func (es Episodes) Less(i, j int) bool {
	if es[i].EpNo.Type == es[j].EpNo.Type {
		if es[i].EpNo.Type == 1 {
			a, _ := strconv.ParseInt(es[i].EpNo.EpNo, 10, 32)
			b, _ := strconv.ParseInt(es[j].EpNo.EpNo, 10, 32)
			return a < b
		} else {
			a, _ := strconv.ParseInt(es[i].EpNo.EpNo[1:], 10, 32)
			b, _ := strconv.ParseInt(es[j].EpNo.EpNo[1:], 10, 32)
			return a < b
		}
	}
	return es[i].EpNo.Type < es[j].EpNo.Type
}

func (es Episodes) Swap(i, j int) {
	es[i], es[j] = es[j], es[i]
}

func (cs Characters) Len() int {
	return len(cs)
}

func (cs Characters) Less(i, j int) bool {
	if cs[i].Type == cs[j].Type {
		return sort.StringSlice{cs[i].Name, cs[j].Name}.Less(0, 1)
	}

	a := 0
	switch cs[i].Type {
	case "main character in":
		a = 0
	case "secondary cast in":
		a = 1
	case "appears in":
		a = 2
	default:
		a = 3
	}

	b := 0
	switch cs[j].Type {
	case "main character in":
		b = 0
	case "secondary cast in":
		b = 1
	case "appears in":
		b = 2
	default:
		b = 3
	}

	return a < b
}

func (cs Characters) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}
