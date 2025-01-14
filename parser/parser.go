package parser

import (
	"github.com/sl2.0/ast"
	"github.com/sl2.0/lexer"
	"github.com/sl2.0/tokens"
)

type (
	prefixFn func() ast.Expression
	infixFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	lexer  *lexer.Lexer
	errors []string

	currentToken tokens.Token
	nextToken    tokens.Token

	infixParseFns  map[tokens.TokenType]infixFn
	prefixParseFns map[tokens.TokenType]prefixFn
}

const (
	LOWEST    = iota
	EQUALS    // ==
	GREATLESS // < >
	SUM       // + -
	PROD      // * /
	PREFIX    // -X  !X
	CALL      // foo(bar)
)

var precedences = map[string]int{
	tokens.EQUALS:   EQUALS,
	tokens.NOTEQUAL: EQUALS,
	tokens.LT:       GREATLESS,
	tokens.GT:       GREATLESS,
	tokens.PLUS:     SUM,
	tokens.MINUS:    SUM,
	tokens.ASTERISC: PROD,
	tokens.SLASH:    PROD,
	tokens.FUNCTION: CALL,
	tokens.LPAR:     CALL,
}

// Generates a new parser using the given input string
func NewParser(input string) *Parser {
	parser := &Parser{
		lexer:  lexer.NewLexer(input),
		errors: []string{},

		infixParseFns:  make(map[tokens.TokenType]infixFn),
		prefixParseFns: make(map[tokens.TokenType]prefixFn),
	}

	parser.InitParsingFns()

	return parser
}

// Returns a new parser using the tokens from a custom lexer
func NewParserFromLexer(lexer *lexer.Lexer) *Parser {
	parser := &Parser{
		lexer:  lexer,
		errors: []string{},

		infixParseFns:  make(map[tokens.TokenType]infixFn),
		prefixParseFns: make(map[tokens.TokenType]prefixFn),
	}

	parser.InitParsingFns()

	return parser
}

func (parser *Parser) InitParsingFns() {
	// to setup the parser in the correct initial state
	parser.advanceToken()
	parser.advanceToken()

	parser.registerPrefixFn(tokens.BANG, parser.parsePrefixExpression)
	parser.registerPrefixFn(tokens.MINUS, parser.parsePrefixExpression)
	parser.registerPrefixFn(tokens.IDENT, parser.parseIdentifier)
	parser.registerPrefixFn(tokens.NUMBER, parser.parseNumber)
	parser.registerPrefixFn(tokens.STRING, parser.parseString)
	parser.registerPrefixFn(tokens.TRUE, parser.parseBoolExpression)
	parser.registerPrefixFn(tokens.FALSE, parser.parseBoolExpression)
	parser.registerPrefixFn(tokens.LPAR, parser.parseGroupedExpression)
	parser.registerPrefixFn(tokens.IF, parser.parseIfExpression)
	parser.registerPrefixFn(tokens.FUNCTION, parser.parseAnonnymousFunction)
	parser.registerPrefixFn(tokens.FOR, parser.parseForLoop)

	parser.registerInfixFn(tokens.MINUS, parser.parseInfixExpression)
	parser.registerInfixFn(tokens.PLUS, parser.parseInfixExpression)
	parser.registerInfixFn(tokens.SLASH, parser.parseInfixExpression)
	parser.registerInfixFn(tokens.ASTERISC, parser.parseInfixExpression)
	parser.registerInfixFn(tokens.GT, parser.parseInfixExpression)
	parser.registerInfixFn(tokens.LT, parser.parseInfixExpression)
	parser.registerInfixFn(tokens.EQUALS, parser.parseInfixExpression)
	parser.registerInfixFn(tokens.NOTEQUAL, parser.parseInfixExpression)
	parser.registerInfixFn(tokens.LPAR, parser.parseCall)
}

func (p *Parser) ParseProgram() *ast.Program {
	tree := &ast.Program{}
	tree.Statements = []ast.Statement{}

	for !p.curTokenIs(tokens.EOF) {
		stmt := p.parseStatement()

		if stmt != nil {
			tree.Statements = append(tree.Statements, stmt)
		}

		p.advanceToken()
	}

	return tree
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.currentToken.Type {
	case tokens.VAR:
		return p.parseVarStatement()
	case tokens.RETURN:
		return p.parseReturnStatement()
	case tokens.FUNCTION:
		return p.parseFunctionStatement()
	case tokens.LINEBREAK:
		return nil
	default:
		return p.parseExpressionStatement()
	}
}

// First parse the prefix side of the expression (identifiers, numbers and unary operators),
// then parse the infix part of the expression if exists
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.currentToken.Type]

	if prefix == nil {
		p.errors = append(p.errors, "Not prefixFn found for: "+p.currentToken.Literal)
		return nil
	}

	exp := prefix()

	for !p.nextTokenIs(tokens.SEMICOLON) && precedence < p.nextPrecendence() {
		infix := p.infixParseFns[p.nextToken.Type]

		if infix == nil {
			return exp
		}

		p.advanceToken()

		// parse and create an infix expression adding the current prefix expression to it
		exp = infix(exp)
	}

	return exp
}
