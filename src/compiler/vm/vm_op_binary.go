package vm

import (
	"ape/src/compiler/operation"
	"ape/src/data"
	"fmt"
)

func (vm *VM) executeBinaryOp(op operation.Opcode) error {
	right := vm.stack.pop()
	left := vm.stack.pop()

	leftType := left.Type()
	rightType := right.Type()

	if leftType == data.INTEGER_TYPE && rightType == data.INTEGER_TYPE {
		return vm.executeBinaryIntegerOp(op, left, right)
	}
	return fmt.Errorf("unsupported types for binary operation: %s %s", leftType, rightType)
}

func (vm *VM) executeBinaryIntegerOp(op operation.Opcode, left, right data.Data) error {
	leftValue := left.(*data.Integer).Value
	rightValue := right.(*data.Integer).Value

	var result int64

	switch op {
	case operation.Add:
		result = leftValue + rightValue

	case operation.Sub:
		result = leftValue - rightValue

	case operation.Mul:
		result = leftValue * rightValue

	case operation.Div:
		result = leftValue / rightValue

	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}
	return vm.stack.push(&data.Integer{Value: result})
}
