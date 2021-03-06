package eval

import (
	"github.com/ape-lang/ape/src/data"
)

func evalPrefixExpression(operator string, right data.Data) data.Data {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return evalError("Unknown operator: %s%s", operator, right.Type())
	}
}
