package database

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/gorm"
)

// --- AST ---

type exprNode struct {
	kind     string // "number", "string", "prop_ref", "binary_op", "if_expr", "func_call"
	number   float64
	text     string
	propName string
	op       string
	left     *exprNode
	right    *exprNode
	cond     *exprNode
	thenExpr *exprNode
	elseExpr *exprNode
	funcName string
	args     []*exprNode
}

// --- Parser ---

type formulaParser struct {
	input []rune
	pos   int
}

func newParser(input string) *formulaParser {
	return &formulaParser{input: []rune(input), pos: 0}
}

func (p *formulaParser) peek() rune {
	if p.pos < len(p.input) {
		return p.input[p.pos]
	}
	return 0
}

func (p *formulaParser) advance() rune {
	ch := p.peek()
	if ch != 0 {
		p.pos++
	}
	return ch
}

func (p *formulaParser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(p.input[p.pos]) {
		p.pos++
	}
}

func (p *formulaParser) parse() (*exprNode, error) {
	return p.parseExpr()
}

func (p *formulaParser) parseExpr() (*exprNode, error) {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	// Parse left side
	left, err := p.parseAtom()
	if err != nil {
		return nil, err
	}

	p.skipWhitespace()

	// Check for binary operator
	op := ""
	switch p.peek() {
	case '+', '-', '*', '/', '=', '!', '>', '<':
		op = p.readOp()
	}

	if op == "" {
		return left, nil
	}

	right, err := p.parseExpr() // right-recursive for correct associativity
	if err != nil {
		return nil, err
	}

	return &exprNode{kind: "binary_op", op: op, left: left, right: right}, nil
}

func (p *formulaParser) readOp() string {
	ch := p.advance()
	switch ch {
	case '+', '-', '*', '/':
		return string(ch)
	case '=':
		return "="
	case '!':
		if p.peek() == '=' {
			p.advance()
			return "!="
		}
		return "!="
	case '>':
		if p.peek() == '=' {
			p.advance()
			return ">="
		}
		return ">"
	case '<':
		if p.peek() == '=' {
			p.advance()
			return "<="
		}
		return "<"
	}
	return ""
}

func (p *formulaParser) parseAtom() (*exprNode, error) {
	p.skipWhitespace()

	ch := p.peek()

	// Number literal
	if ch == '.' || (ch >= '0' && ch <= '9') || ch == '-' {
		// Check if it's a negative number (no binary op context — if preceded by op, parseExpr handles it)
		if ch == '-' || ch == '.' || (ch >= '0' && ch <= '9') {
			return p.parseNumber()
		}
	}

	// String literal
	if ch == '"' {
		return p.parseString()
	}

	// Parenthesized expression
	if ch == '(' {
		p.advance()
		node, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.advance() != ')' {
			return nil, fmt.Errorf("expected ')'")
		}
		return node, nil
	}

	// Function call or prop_ref or if
	if isIdentStart(ch) {
		ident := p.readIdent()

		p.skipWhitespace()
		if p.peek() == '(' {
			p.advance() // consume '('

			switch ident {
			case "prop":
				return p.parsePropRef()
			case "if":
				return p.parseIfExpr()
			default:
				return p.parseFuncCall(ident)
			}
		}
		return nil, fmt.Errorf("unknown identifier: %s", ident)
	}

	return nil, fmt.Errorf("unexpected character: %c at position %d", ch, p.pos)
}

func (p *formulaParser) parseNumber() (*exprNode, error) {
	start := p.pos
	// Allow leading minus
	if p.peek() == '-' {
		p.advance()
	}
	// Digits before decimal
	for p.peek() >= '0' && p.peek() <= '9' {
		p.advance()
	}
	// Decimal part
	if p.peek() == '.' {
		p.advance()
		for p.peek() >= '0' && p.peek() <= '9' {
			p.advance()
		}
	}
	text := string(p.input[start:p.pos])
	n, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number: %s", text)
	}
	return &exprNode{kind: "number", number: n}, nil
}

func (p *formulaParser) parseString() (*exprNode, error) {
	p.advance() // consume opening "
	var sb strings.Builder
	for {
		ch := p.advance()
		if ch == 0 {
			return nil, fmt.Errorf("unterminated string")
		}
		if ch == '"' {
			break
		}
		if ch == '\\' {
			next := p.advance()
			switch next {
			case '"':
				sb.WriteRune('"')
			case '\\':
				sb.WriteRune('\\')
			case 'n':
				sb.WriteRune('\n')
			default:
				sb.WriteRune(ch)
				sb.WriteRune(next)
			}
		} else {
			sb.WriteRune(ch)
		}
	}
	return &exprNode{kind: "string", text: sb.String()}, nil
}

func (p *formulaParser) parsePropRef() (*exprNode, error) {
	// We've already consumed "prop("
	p.skipWhitespace()
	ch := p.peek()
	if ch != '"' {
		return nil, fmt.Errorf("prop() expects a string argument")
	}
	propNode, err := p.parseString()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.advance() != ')' {
		return nil, fmt.Errorf("expected ')' after prop()")
	}
	return &exprNode{kind: "prop_ref", propName: propNode.text}, nil
}

func (p *formulaParser) parseIfExpr() (*exprNode, error) {
	// We've already consumed "if("
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.advance() != ',' {
		return nil, fmt.Errorf("expected ',' in if()")
	}
	thenExpr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.advance() != ',' {
		return nil, fmt.Errorf("expected ',' in if()")
	}
	elseExpr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.advance() != ')' {
		return nil, fmt.Errorf("expected ')' after if()")
	}
	return &exprNode{kind: "if_expr", cond: cond, thenExpr: thenExpr, elseExpr: elseExpr}, nil
}

func (p *formulaParser) parseFuncCall(name string) (*exprNode, error) {
	// We've already consumed "name("
	var args []*exprNode

	p.skipWhitespace()
	if p.peek() == ')' {
		p.advance()
		return &exprNode{kind: "func_call", funcName: name, args: args}, nil
	}

	for {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		p.skipWhitespace()
		ch := p.advance()
		if ch == ')' {
			break
		}
		if ch != ',' {
			return nil, fmt.Errorf("expected ',' or ')' in function call, got %c", ch)
		}
	}
	return &exprNode{kind: "func_call", funcName: name, args: args}, nil
}

func (p *formulaParser) readIdent() string {
	start := p.pos
	for isIdentPart(p.peek()) {
		p.advance()
	}
	return string(p.input[start:p.pos])
}

func isIdentStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || (r >= '0' && r <= '9')
}

// --- Evaluator ---

type formulaEvaluator struct {
	DB           *gorm.DB
	databaseID   uint
	recordID     uint
	propertyMap  map[string]uint // name → id
	propertyVals map[uint]string // id → JSONB value
}

func (e *formulaEvaluator) eval(node *exprNode) (any, error) {
	switch node.kind {
	case "number":
		return node.number, nil
	case "string":
		return node.text, nil
	case "prop_ref":
		return e.evalPropRef(node.propName)
	case "binary_op":
		return e.evalBinaryOp(node)
	case "if_expr":
		return e.evalIf(node)
	case "func_call":
		return e.evalFuncCall(node)
	default:
		return nil, fmt.Errorf("unknown node kind: %s", node.kind)
	}
}

func (e *formulaEvaluator) evalPropRef(name string) (any, error) {
	propID, ok := e.propertyMap[name]
	if !ok {
		return "", nil // property not found → empty
	}
	valJSON, ok := e.propertyVals[propID]
	if !ok || valJSON == "" || valJSON == "{}" {
		return "", nil
	}
	var v map[string]interface{}
	if err := json.Unmarshal([]byte(valJSON), &v); err != nil {
		return "", nil
	}
	// Return first meaningful value
	if n, ok := v["number"]; ok {
		switch num := n.(type) {
		case float64:
			return num, nil
		}
	}
	if t, ok := v["text"]; ok {
		if ts, isStr := t.(string); isStr {
			return ts, nil
		}
	}
	if s, ok := v["select"]; ok {
		if ss, isStr := s.(string); isStr {
			return ss, nil
		}
	}
	if d, ok := v["date"]; ok {
		if ds, isStr := d.(string); isStr {
			return ds, nil
		}
	}
	if c, ok := v["checked"]; ok {
		if cb, isBool := c.(bool); isBool {
			return cb, nil
		}
	}
	return "", nil
}

func (e *formulaEvaluator) evalBinaryOp(node *exprNode) (any, error) {
	left, err := e.eval(node.left)
	if err != nil {
		return nil, err
	}
	right, err := e.eval(node.right)
	if err != nil {
		return nil, err
	}

	// For addition, if either is string, do string concat
	if node.op == "+" {
		_, lIsStr := left.(string)
		_, rIsStr := right.(string)
		if lIsStr || rIsStr {
			return fmt.Sprintf("%v%v", left, right), nil
		}
	}

	lNum, lIsNum := toFloat(left)
	rNum, rIsNum := toFloat(right)

	if lIsNum && rIsNum {
		switch node.op {
		case "+":
			return lNum + rNum, nil
		case "-":
			return lNum - rNum, nil
		case "*":
			return lNum * rNum, nil
		case "/":
			if rNum == 0 {
				return float64(0), nil
			}
			return lNum / rNum, nil
		case "=":
			return lNum == rNum, nil
		case "!=":
			return lNum != rNum, nil
		case ">":
			return lNum > rNum, nil
		case "<":
			return lNum < rNum, nil
		case ">=":
			return lNum >= rNum, nil
		case "<=":
			return lNum <= rNum, nil
		}
	}

	// Comparison of non-numeric values
	switch node.op {
	case "=":
		return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right), nil
	case "!=":
		return fmt.Sprintf("%v", left) != fmt.Sprintf("%v", right), nil
	}

	return "", nil
}

func (e *formulaEvaluator) evalIf(node *exprNode) (any, error) {
	cond, err := e.eval(node.cond)
	if err != nil {
		return nil, err
	}
	if isTruthy(cond) {
		return e.eval(node.thenExpr)
	}
	return e.eval(node.elseExpr)
}

func (e *formulaEvaluator) evalFuncCall(node *exprNode) (any, error) {
	switch node.funcName {
	case "concat":
		var parts []string
		for _, arg := range node.args {
			val, err := e.eval(arg)
			if err != nil {
				return nil, err
			}
			parts = append(parts, fmt.Sprintf("%v", val))
		}
		return strings.Join(parts, ""), nil

	case "lower":
		if len(node.args) != 1 {
			return nil, fmt.Errorf("lower() takes 1 argument")
		}
		val, err := e.eval(node.args[0])
		if err != nil {
			return nil, err
		}
		return strings.ToLower(fmt.Sprintf("%v", val)), nil

	case "upper":
		if len(node.args) != 1 {
			return nil, fmt.Errorf("upper() takes 1 argument")
		}
		val, err := e.eval(node.args[0])
		if err != nil {
			return nil, err
		}
		return strings.ToUpper(fmt.Sprintf("%v", val)), nil

	case "length":
		if len(node.args) != 1 {
			return nil, fmt.Errorf("length() takes 1 argument")
		}
		val, err := e.eval(node.args[0])
		if err != nil {
			return nil, err
		}
		return float64(len(fmt.Sprintf("%v", val))), nil

	default:
		return nil, fmt.Errorf("unknown function: %s", node.funcName)
	}
}

// --- FormulaService ---

type FormulaService struct {
	DB *gorm.DB
}

func NewFormulaService(d *gorm.DB) *FormulaService {
	return &FormulaService{DB: d}
}

// ComputeFormula parses the expression and evaluates it against the record's
// property values. Returns a JSONB value string.
func (s *FormulaService) ComputeFormula(databaseID, recordID uint, expression string) (string, error) {
	parser := newParser(expression)
	ast, err := parser.parse()
	if err != nil {
		return fmt.Sprintf(`{"text": "#ERROR: %s"}`, err.Error()), nil
	}

	// Load property map (name → id) for this database
	var properties []db.Property
	s.DB.Where("database_id = ?", databaseID).Find(&properties)
	propMap := make(map[string]uint)
	for _, p := range properties {
		propMap[p.Name] = p.ID
	}

	// Load property values for this record
	var pvs []db.PropertyValue
	s.DB.Where("record_id = ?", recordID).Find(&pvs)
	propVals := make(map[uint]string)
	for _, pv := range pvs {
		propVals[pv.PropertyID] = pv.Value
	}

	evaluator := &formulaEvaluator{
		DB:           s.DB,
		databaseID:   databaseID,
		recordID:     recordID,
		propertyMap:  propMap,
		propertyVals: propVals,
	}

	result, err := evaluator.eval(ast)
	if err != nil {
		return fmt.Sprintf(`{"text": "#ERROR: %s"}`, err.Error()), nil
	}

	switch v := result.(type) {
	case float64:
		return fmt.Sprintf(`{"number": %f}`, v), nil
	case bool:
		if v {
			return `{"text": "true"}`, nil
		}
		return `{"text": "false"}`, nil
	default:
		return fmt.Sprintf(`{"text": "%s"}`, escapeJSON(fmt.Sprintf("%v", v))), nil
	}
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}

func isTruthy(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case string:
		return val != ""
	default:
		return false
	}
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
