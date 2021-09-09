package tlps

import (
	"fmt"
	"reflect"
)

type Interpreter struct {
	Runtime *Runtime
}

func NewInterpreter(runtime *Runtime) *Interpreter {
	globals := runtime.Globals

	globals.Define("clock", NewClockFunc())

	return &Interpreter{
		Runtime: runtime,
	}
}

func (i *Interpreter) Interpret(statements []Stmt) (string, error) {
	var s string
	var err error
	for _, statement := range statements {
		v, er := i.execute(statement)
		err = er
		s = stringfy(v)
		if err != nil {
			i.Runtime.RuntimeError(err)
		}
	}

	return s, err
}

func (i *Interpreter) visitBinaryExpr(expr *Binary) (interface{}, error) {
	left, err := i.evaluate(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := i.evaluate(expr.Right)
	if err != nil {
		return nil, err
	}

	switch expr.Operator.Type {
	case Greater:
		err := checkNumberOperands(expr.Operator, left, right)
		if err != nil {
			return nil, err
		}
		return left.(float64) > right.(float64), nil
	case GreaterEqual:
		err := checkNumberOperands(expr.Operator, left, right)
		if err != nil {
			return nil, err
		}
		return left.(float64) >= right.(float64), nil
	case Less:
		err := checkNumberOperands(expr.Operator, left, right)
		if err != nil {
			return nil, err
		}
		return left.(float64) < right.(float64), nil
	case LessEqual:
		err := checkNumberOperands(expr.Operator, left, right)
		if err != nil {
			return nil, err
		}
		return left.(float64) <= right.(float64), nil
	case BangEqual:
		err := checkNumberOperands(expr.Operator, left, right)
		if err != nil {
			return nil, err
		}
		return left.(float64) != right.(float64), nil
	case EqualEqual:
		err := checkNumberOperands(expr.Operator, left, right)
		if err != nil {
			return nil, err
		}
		return left.(float64) == right.(float64), nil
	case Minus:
		err := checkNumberOperands(expr.Operator, left, right)
		if err != nil {
			return nil, err
		}
		return left.(float64) - right.(float64), nil
	case Plus:
		if isFloat64(left) && isFloat64(right) {
			return left.(float64) + right.(float64), nil
		}
		if isString(left) && isString(right) {
			return left.(string) + right.(string), nil
		}

		return nil, RuntimeError.New(expr.Operator, "Operands must be two numbers or two strings.")
	case Slash:
		err := checkNumberOperands(expr.Operator, left, right)
		if err != nil {
			return nil, err
		}
		return left.(float64) / right.(float64), nil
	case Star:
		err := checkNumberOperands(expr.Operator, left, right)
		if err != nil {
			return nil, err
		}
		return left.(float64) * right.(float64), nil
	}

	// Unreachable.
	return nil, RuntimeError.New(nil, "Unreachable")
}

func (i *Interpreter) visitCallExpr(expr *Call) (interface{}, error) {
	callee, err := i.evaluate(expr.Callee)
	if err != nil {
		return nil, err
	}

	arguments := make([]interface{}, 0)
	for _, argument := range expr.Arguments {
		arg, err := i.evaluate(argument)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, arg)
	}

	function, ok := callee.(LoxCallable)
	if !ok {
		return nil, RuntimeError.New(expr.Paren, "Can only call functions and classes.")
	}

	if len(arguments) != function.Arity() {
		return nil, RuntimeError.New(expr.Paren, fmt.Sprintf("Expected %d arguments but got %d.", function.Arity(), len(arguments)))
	}

	return function.Call(i, arguments)
}

func (i *Interpreter) visitLiteralExpr(expr *Literal) (interface{}, error) {
	return expr.Value, nil
}

func (i *Interpreter) visitLogicalExpr(expr *Logical) (interface{}, error) {
	left, err := i.evaluate(expr.Left)
	if err != nil {
		return nil, err
	}

	if expr.Operator.Type == Or {
		if i.isTruthy(left) {
			return left, nil
		}
	} else {
		if !i.isTruthy(left) {
			return left, nil
		}
	}

	return i.evaluate(expr.Right)
}

func (i *Interpreter) visitGroupingExpr(expr *Grouping) (interface{}, error) {
	return i.evaluate(expr.Expression)
}

func (i *Interpreter) visitUnaryExpr(expr *Unary) (interface{}, error) {
	right, err := i.evaluate(expr.Right)
	if err != nil {
		return nil, err
	}

	switch expr.Operator.Type {
	case Bang:
		return !i.isTruthy(right), nil
	case Minus:
		err := checkNumberOperand(expr.Operator, right)
		if err != nil {
			return nil, err
		}
		return -right.(float64), nil
	}

	// Unreachable
	return nil, RuntimeError.New(nil, "Unreachable")
}

func (i *Interpreter) visitVariableExpr(expr *Variable) (interface{}, error) {
	return i.Runtime.Environment.Get(expr.Name)
}

func checkNumberOperand(operator *Token, operand interface{}) error {
	if reflect.ValueOf(operand).Kind() == reflect.Float64 {
		return nil
	}
	return RuntimeError.New(operator, "Operand must be a number.")
}

func checkNumberOperands(operator *Token, left interface{}, right interface{}) error {
	if reflect.ValueOf(left).Kind() == reflect.Float64 &&
		reflect.ValueOf(right).Kind() == reflect.Float64 {
		return nil
	}
	return RuntimeError.New(operator, "Operands must be a number.")
}

func (i *Interpreter) isTruthy(object interface{}) bool {
	if object == nil {
		return false
	}
	switch v := reflect.ValueOf(object); v.Kind() {
	case reflect.Bool:
		return object.(bool)
	}

	return true
}

func (i *Interpreter) evaluate(expr Expr) (interface{}, error) {
	return expr.Accept(i)
}

func (i *Interpreter) execute(stmt Stmt) (interface{}, error) {
	return stmt.Accept(i)
}

func (i *Interpreter) executeBlock(statements []Stmt, environment *Environment) (interface{}, error) {
	previous := i.Runtime.Environment
	defer func() { i.Runtime.Environment = previous }()
	i.Runtime.Environment = environment
	for _, statement := range statements {
		_, err := i.execute(statement)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (i *Interpreter) visitBlockStmt(stmt *Block) (interface{}, error) {
	return i.executeBlock(stmt.Statements, NewEnvironment(i.Runtime.Environment))
}

func isEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		return false
	}

	return a == b
}

func (i *Interpreter) visitExpressionStmt(stmt *Expression) (interface{}, error) {
	return i.evaluate(stmt.Expression)
	// return nil, nil
}

func (i *Interpreter) visitFunctionStmt(stmt *Function) (interface{}, error) {
	function := NewLoxFunction(stmt, i.Runtime.Environment)
	i.Runtime.Environment.Define(stmt.Name.Lexeme, function)
	return nil, nil
}

func (i *Interpreter) visitIf_Stmt(stmt *If_) (interface{}, error) {
	val, err := i.evaluate(stmt.Condition)
	if err != nil {
		return nil, err
	}
	if i.isTruthy(val) {
		return i.execute(stmt.ThenBranch)
	} else if stmt.ElseBranch != nil {
		return i.execute(stmt.ElseBranch)
	}
	return nil, nil
}

func (i *Interpreter) visitPrint_Stmt(stmt *Print_) (interface{}, error) {
	value, err := i.evaluate(stmt.Expression)
	if err != nil {
		return nil, err
	}

	fmt.Println(stringfy(value))
	return nil, nil
}

func (i *Interpreter) visitReturn_Stmt(stmt *Return_) (interface{}, error) {
	var value interface{} = nil
	if stmt.Value != nil {
		var err error
		value, err = i.evaluate(stmt.Value)
		if err != nil {
			return nil, err
		}
	}

	return nil, NewReturnValue(value)
}

func (i *Interpreter) visitWhile_Stmt(stmt *While_) (interface{}, error) {
	v, _ := i.evaluate(stmt.Condition)
	for ; i.isTruthy(v); v, _ = i.evaluate(stmt.Condition) {
		i.execute(stmt.Body)
	}

	return nil, nil
}

func (i *Interpreter) visitVar_Stmt(stmt *Var_) (interface{}, error) {
	var value interface{} = nil
	if stmt.Initializer != nil {
		v, err := i.evaluate(stmt.Initializer)
		value = v
		if err != nil {
			return nil, err
		}
	}

	i.Runtime.Environment.Define(stmt.Name.Lexeme, value)
	return nil, nil
}

func (i *Interpreter) visitAssignExpr(expr *Assign) (interface{}, error) {
	value, err := i.evaluate(expr.Value)
	if err != nil {
		return nil, err
	}
	err = i.Runtime.Environment.Assign(expr.Name, value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// ReturnValue is struct of return value
type ReturnValue struct {
	Value interface{}
}

// Error satisfies error interface
func (r *ReturnValue) Error() string {
	return "Return Value error"
}

// NewReturnValue is constructor of ReturValue
func NewReturnValue(value interface{}) *ReturnValue {
	return &ReturnValue{Value: value}
}

func isType(v interface{}, kind reflect.Kind) bool {
	return reflect.ValueOf(v).Kind() == kind
}

func isFloat64(v interface{}) bool {
	return isType(v, reflect.Float64)
}

func isString(v interface{}) bool {
	return isType(v, reflect.String)
}

func stringfy(object interface{}) string {
	if object == nil {
		return "nil"
	}

	return fmt.Sprint(object)
}
