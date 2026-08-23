// Package filter 提供白名单 AST 标量过滤，禁止 eval。
package filter

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xavskye/gorag/internal/model"
)

type Kind int

const (
	KindAND Kind = iota
	KindOR
	KindCmp
)

type Op string

const (
	OpEq Op = "=="
	OpNe Op = "!="
	OpGt Op = ">"
	OpGe Op = ">="
	OpLt Op = "<"
	OpLe Op = "<="
)

var allowedFields = map[string]struct{}{
	"tag": {}, "tags": {}, "caption": {}, "modality": {}, "score": {},
	"collection": {}, "doc_id": {}, "source_ref": {},
}

type Node struct {
	Kind  Kind
	Left  *Node
	Right *Node
	Field string
	Op    Op
	Value string
	Num   float64
	IsNum bool
}

func Parse(expr string) (*Node, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, nil
	}
	p := &parser{s: expr}
	n, err := p.parseOr()
	if err != nil {
		return nil, model.Wrap(model.CodeFilterInvalid, "invalid filter", err)
	}
	p.skip()
	if p.pos < len(p.s) {
		return nil, model.NewError(model.CodeFilterInvalid, "trailing tokens in filter")
	}
	return n, nil
}

type parser struct {
	s   string
	pos int
}

func (p *parser) skip() {
	for p.pos < len(p.s) {
		r, w := utf8.DecodeRuneInString(p.s[p.pos:])
		if !unicode.IsSpace(r) {
			return
		}
		p.pos += w
	}
}

func (p *parser) peek() string {
	p.skip()
	if p.pos >= len(p.s) {
		return ""
	}
	return p.s[p.pos:]
}

func (p *parser) parseOr() (*Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		rest := strings.TrimSpace(p.peek())
		if !strings.HasPrefix(strings.ToLower(rest), "||") && !strings.HasPrefix(strings.ToLower(rest), "or ") {
			if strings.HasPrefix(strings.ToLower(rest), "or") && (len(rest) == 2 || !isIdent(rune(rest[2]))) {
				p.pos += 2
			} else {
				break
			}
		} else if strings.HasPrefix(rest, "||") {
			p.pos += 2
		} else {
			p.pos += 2
		}
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &Node{Kind: KindOR, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (*Node, error) {
	left, err := p.parseCmp()
	if err != nil {
		return nil, err
	}
	for {
		rest := p.peek()
		low := strings.ToLower(strings.TrimSpace(rest))
		switch {
		case strings.HasPrefix(low, "&&"):
			p.skip()
			p.pos += 2
		case strings.HasPrefix(low, "and") && (len(low) == 3 || !isIdent(rune(low[3]))):
			p.skip()
			p.pos += 3
		default:
			return left, nil
		}
		right, err := p.parseCmp()
		if err != nil {
			return nil, err
		}
		left = &Node{Kind: KindAND, Left: left, Right: right}
	}
}

func (p *parser) parseCmp() (*Node, error) {
	p.skip()
	if p.pos < len(p.s) && p.s[p.pos] == '(' {
		p.pos++
		n, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		p.skip()
		if p.pos >= len(p.s) || p.s[p.pos] != ')' {
			return nil, fmt.Errorf("missing )")
		}
		p.pos++
		return n, nil
	}
	field, err := p.ident()
	if err != nil {
		return nil, err
	}
	if _, ok := allowedFields[field]; !ok {
		return nil, fmt.Errorf("field %q not in whitelist", field)
	}
	op, err := p.op()
	if err != nil {
		return nil, err
	}
	val, num, isNum, err := p.literal()
	if err != nil {
		return nil, err
	}
	return &Node{Kind: KindCmp, Field: field, Op: op, Value: val, Num: num, IsNum: isNum}, nil
}

func (p *parser) ident() (string, error) {
	p.skip()
	start := p.pos
	for p.pos < len(p.s) {
		r, w := utf8.DecodeRuneInString(p.s[p.pos:])
		if !isIdent(r) && r != '_' {
			break
		}
		p.pos += w
	}
	if p.pos == start {
		return "", fmt.Errorf("expected field")
	}
	return strings.ToLower(p.s[start:p.pos]), nil
}

func (p *parser) op() (Op, error) {
	p.skip()
	for _, cand := range []string{">=", "<=", "==", "!=", ">", "<"} {
		if strings.HasPrefix(p.s[p.pos:], cand) {
			p.pos += len(cand)
			return Op(cand), nil
		}
	}
	return "", fmt.Errorf("expected comparator")
}

func (p *parser) literal() (string, float64, bool, error) {
	p.skip()
	if p.pos >= len(p.s) {
		return "", 0, false, fmt.Errorf("expected literal")
	}
	if p.s[p.pos] == '"' || p.s[p.pos] == '\'' {
		q := p.s[p.pos]
		p.pos++
		start := p.pos
		for p.pos < len(p.s) && p.s[p.pos] != q {
			p.pos++
		}
		if p.pos >= len(p.s) {
			return "", 0, false, fmt.Errorf("unterminated string")
		}
		val := p.s[start:p.pos]
		p.pos++
		return val, 0, false, nil
	}
	start := p.pos
	if p.s[p.pos] == '-' {
		p.pos++
	}
	for p.pos < len(p.s) {
		r := p.s[p.pos]
		if (r >= '0' && r <= '9') || r == '.' {
			p.pos++
			continue
		}
		break
	}
	raw := p.s[start:p.pos]
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return raw, 0, false, nil
	}
	return raw, n, true, nil
}

func isIdent(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
