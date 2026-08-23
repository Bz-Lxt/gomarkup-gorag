package filter

import (
	"strconv"
	"strings"

	"github.com/xavskye/gorag/internal/model"
)

func Match(n *Node, e *model.Entity, score float64) bool {
	if n == nil {
		return true
	}
	switch n.Kind {
	case KindAND:
		return Match(n.Left, e, score) && Match(n.Right, e, score)
	case KindOR:
		return Match(n.Left, e, score) || Match(n.Right, e, score)
	default:
		return cmp(n, e, score)
	}
}

func cmp(n *Node, e *model.Entity, score float64) bool {
	switch n.Field {
	case "tag", "tags":
		for _, t := range e.Tags {
			if compareStr(t, n.Op, n.Value) {
				return true
			}
		}
		if e.Scalar != nil {
			if v, ok := e.Scalar["tag"]; ok {
				if compareStr(stringify(v), n.Op, n.Value) {
					return true
				}
			}
		}
		return false
	case "caption":
		return compareStr(e.Caption, n.Op, n.Value) || compareStr(e.Content, n.Op, n.Value)
	case "modality":
		return compareStr(string(e.Modality), n.Op, n.Value)
	case "collection":
		return compareStr(e.Collection, n.Op, n.Value)
	case "doc_id":
		return compareStr(e.DocID, n.Op, n.Value)
	case "source_ref":
		return compareStr(e.SourceRef, n.Op, n.Value)
	case "score":
		return compareNum(score, n.Op, n.Num)
	default:
		if e.Scalar != nil {
			if v, ok := e.Scalar[n.Field]; ok {
				if n.IsNum {
					f, err := toFloat(v)
					if err == nil {
						return compareNum(f, n.Op, n.Num)
					}
				}
				return compareStr(stringify(v), n.Op, n.Value)
			}
		}
		return false
	}
}

func compareStr(got string, op Op, want string) bool {
	g := strings.ToLower(strings.TrimSpace(got))
	w := strings.ToLower(strings.TrimSpace(want))
	switch op {
	case OpEq:
		return g == w
	case OpNe:
		return g != w
	default:
		return strings.Contains(g, w)
	}
}

func compareNum(got float64, op Op, want float64) bool {
	switch op {
	case OpEq:
		return got == want
	case OpNe:
		return got != want
	case OpGt:
		return got > want
	case OpGe:
		return got >= want
	case OpLt:
		return got < want
	case OpLe:
		return got <= want
	}
	return false
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	default:
		return ""
	}
}

func toFloat(v any) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case int:
		return float64(t), nil
	case string:
		return strconv.ParseFloat(t, 64)
	default:
		return 0, strconv.ErrSyntax
	}
}
