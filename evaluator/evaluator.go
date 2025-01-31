package evaluator

import (
	"github.com/sl2.0/ast"
	"github.com/sl2.0/objects"
	"github.com/sl2.0/parser"
	"github.com/sl2.0/tokens"
)

var (
	true_obj  = &objects.Boolean{Value: true}
	false_obj = &objects.Boolean{Value: false}
)

type Evaluator struct {
	errors  []string
	program *ast.Program
}

func NewFromInput(input string) *Evaluator {
	eval := &Evaluator{}
	pars := parser.NewParser(input)

	if pars == nil {
		eval.errors = append(eval.errors, "Parser returned a nil value")
		return nil
	}

	eval.program = pars.ParseProgram()

	if pars.HasErrors() {
		eval.errors = pars.Errors()
		return nil
	}

	return eval
}

func NewFromProgram(ast *ast.Program) *Evaluator {
	eval := &Evaluator{}

	if ast == nil {
		eval.errors = append(eval.errors, "Submited an empty(nil) ast")
		return nil
	}

	eval.program = ast

	return eval
}

func (e *Evaluator) Errors() []string {
	return e.errors
}

func (e *Evaluator) HasErrors() bool {
	return len(e.errors) != 0
}

func (e *Evaluator) EvalProgram(env *objects.Storage) objects.Object {
	return e.eval(e.program, env)
}

/*
eval evaluates every statement or expression within the program recursivelly

eval recieves an storage environment, which is local to the execution scope.
So there are no global values. For statements scope dependant (like functions or for loops),
a new env has to be created an passed to the eval function.
*/
func (e *Evaluator) eval(node ast.Node, env *objects.Storage) objects.Object {
	switch node := node.(type) {
	case *ast.Program:
		return e.evalStatements(node.Statements, env)

		// -- Statements --
	case *ast.ExpressionStatement:
		return e.eval(node.Expression, env)

	case *ast.VarStatement:
		val := e.eval(node.Value, env)
		if isError(val) {
			return val
		}

		return env.Set(node.Identifier.Value, val)

	case *ast.Identifier:
		val, ok := env.Get(node.Value)
		if !ok {
			return objects.NewError("Cannot resolve identifier: %s", node.Value)
		}
		return val

	case *ast.FunctionStatement:
		f := &objects.FunctionObject{
			Parameters: node.Parameters,
			Body:       node.Body,
		}

		env.Set(node.Identifier.Value, f)

		return f

	case *ast.AnonymousFunction:
		f := &objects.FunctionObject{
			Parameters: node.Parameters,
			Body:       node.Body,
		}
		return f

	case *ast.FunctionCall:
		return e.evalFunctionCall(node, env)

	case *ast.BlockStatement:
		return e.evalBlockStatement(node, env)

	case *ast.ForLoop:
		return e.evalForLoop(node, env)

	case *ast.ReturnStatement:
		val := e.eval(node.ReturnValue, env)
		return &objects.ReturnObject{Value: val}

		// -- Expressions --
	case *ast.PrefixExpression:
		return e.evalPrefix(node, env)

	case *ast.InfixExpression:
		return e.evalInfix(node, env)

	case *ast.IfExpression:
		return e.evalIfExpression(node, env)

	case *ast.IntegerLiteral:
		return &objects.Integer{Value: node.Value}

	case *ast.Boolean:
		if node.Token.Type == tokens.TRUE {
			return true_obj
		}
		return false_obj

	case *ast.StringLiteral:
		return &objects.String{Value: node.Value}
	}

	return objects.NewError("Cannot evaluate node: %s", node.ToString(0))
}

func (e *Evaluator) evalBlockStatement(node *ast.BlockStatement, env *objects.Storage) objects.Object {
	var res objects.Object

	for _, value := range node.Statements {
		res = e.eval(value, env)

		if res != nil {
			rt := res.Type()
			if rt == objects.RETURN_OBJ || rt == objects.ERROR_OBJ {
				return res
			}
		}
	}

	return res
}

func (e *Evaluator) evalStatements(stmts []ast.Statement, env *objects.Storage) objects.Object {
	var res objects.Object

	for _, value := range stmts {
		res = e.eval(value, env)

		switch res := res.(type) {
		case *objects.ReturnObject:
			return res.Value

		case *objects.ErrorObject:
			return res
		}
	}

	return res
}

func (e *Evaluator) evalExpressions(exps []ast.Expression, env *objects.Storage) []objects.Object {
	var res []objects.Object

	for _, value := range exps {
		r := e.eval(value, env)
		if isError(r) {
			return []objects.Object{r}
		}

		res = append(res, r)
	}

	return res
}

func isError(obj objects.Object) bool {
	if obj != nil {
		rt := obj.Type()
		return rt == objects.ERROR_OBJ
	}

	return false
}

func isReturn(obj objects.Object) bool {
	if obj != nil {
		rt := obj.Type()
		return rt == objects.RETURN_OBJ
	}

	return false
}
