package vm

import (
	"fmt"

	"github.com/ape-lang/ape/src/compiler/compiler"
	"github.com/ape-lang/ape/src/compiler/operation"
	"github.com/ape-lang/ape/src/data"
)

const GlobalsLimit = 65536 // equal to the max value represented by uint16 (operation.Constant)
const StackLimit = 2048
const FrameLimit = 1024

// Global references, so a new object does not get allocated for each evaluation
var (
	TRUE  = &data.Boolean{Value: true}
	FALSE = &data.Boolean{Value: false}
	NULL  = &data.Null{}
)

// VM contains the definition of the VM
type VM struct {
	constants []data.Data
	globals   []data.Data
	stack     *Stack
	frames    *Frames
}

// New creates a new VM from the given Bytecode
func New(bytecode *compiler.Bytecode) *VM {
	// create an execution frame for the main function
	mainFn := &data.CompiledFunction{Instructions: bytecode.Instructions}
	mainFrame := NewFrame(mainFn, 0)

	frames := NewFrames(FrameLimit)
	frames.push(mainFrame)

	return &VM{
		constants: bytecode.Constants,
		globals:   make([]data.Data, GlobalsLimit),
		stack:     NewStack(StackLimit),
		frames:    frames,
	}
}

// NewWithGlobals creates a new VM instance with closure over a globals array (for persistance)
func NewWithGlobals(bytecode *compiler.Bytecode, globals []data.Data) *VM {
	vm := New(bytecode)
	vm.globals = globals
	return vm
}

// Result returns the value of the last popped element from the stack (last evaluated expression)
func (vm *VM) Result() data.Data {
	return vm.stack.popped()
}

// Run executes every instruction given to the VM on creation
func (vm *VM) Run() error {
	var pointer int
	var instructions operation.Instruction
	var op operation.Opcode

	for vm.frames.current().pointer < len(vm.frames.current().Instructions())-1 {
		vm.frames.current().pointer++

		pointer = vm.frames.current().pointer
		instructions = vm.frames.current().Instructions()
		op = operation.Opcode(instructions[pointer])

		switch op {
		case operation.Pop:
			vm.stack.pop()

		case operation.Constant:
			constIndex := operation.ReadUint16(instructions[pointer+1:])
			vm.frames.current().pointer += 2

			err := vm.stack.push(vm.constants[constIndex])
			if err != nil {
				return err
			}

		case operation.True:
			err := vm.stack.push(TRUE)
			if err != nil {
				return err
			}

		case operation.False:
			err := vm.stack.push(FALSE)
			if err != nil {
				return err
			}

		case operation.Bang:
			err := vm.executeBangOp()
			if err != nil {
				return err
			}

		case operation.Minus:
			err := vm.executeMinusOp()
			if err != nil {
				return err
			}

		case operation.Add, operation.Sub, operation.Mul, operation.Div:
			err := vm.executeBinaryOp(op)
			if err != nil {
				return err
			}

		case operation.Equal, operation.NotEqual, operation.GreaterThan:
			err := vm.executeComparison(op)
			if err != nil {
				return err
			}

		case operation.Jump:
			pos := int(operation.ReadUint16(instructions[pointer+1:]))
			vm.frames.current().pointer = pos - 1

		case operation.JumpNotTruthy:
			pos := int(operation.ReadUint16(instructions[pointer+1:]))
			vm.frames.current().pointer += 2
			condition := vm.stack.pop()
			if !isTruthy(condition) {
				vm.frames.current().pointer = pos - 1
			}

		case operation.Null:
			err := vm.stack.push(NULL)
			if err != nil {
				return err
			}

		case operation.SetGlobal:
			index := operation.ReadUint16(instructions[pointer+1:])
			vm.frames.current().pointer += 2
			vm.globals[index] = vm.stack.pop()

		case operation.GetGlobal:
			index := operation.ReadUint16(instructions[pointer+1:])
			vm.frames.current().pointer += 2
			err := vm.stack.push(vm.globals[index])
			if err != nil {
				return err
			}

		case operation.SetLocal:
			localIndex := operation.ReadUint8(instructions[pointer+1:])
			vm.frames.current().pointer++
			frame := vm.frames.current()
			vm.stack.items[frame.framePointer+int(localIndex)] = vm.stack.pop()

		case operation.GetLocal:
			localIndex := operation.ReadUint8(instructions[pointer+1:])
			vm.frames.current().pointer++
			frame := vm.frames.current()
			err := vm.stack.push(vm.stack.items[frame.framePointer+int(localIndex)])
			if err != nil {
				return err
			}

		case operation.Array:
			numElements := int(operation.ReadUint16(instructions[pointer+1:]))
			vm.frames.current().pointer += 2
			array := vm.buildArray(vm.stack.pointer-numElements, vm.stack.pointer)
			vm.stack.pointer = vm.stack.pointer - numElements
			err := vm.stack.push(array)
			if err != nil {
				return err
			}

		case operation.Hash:
			numElements := int(operation.ReadUint16(instructions[pointer+1:]))
			vm.frames.current().pointer += 2
			hash, err := vm.buildHash(vm.stack.pointer-numElements, vm.stack.pointer)
			if err != nil {
				return err
			}
			vm.stack.pointer = vm.stack.pointer - numElements
			err = vm.stack.push(hash)
			if err != nil {
				return err
			}

		case operation.Index:
			index := vm.stack.pop()
			left := vm.stack.pop()
			err := vm.executeIndexExpr(left, index)
			if err != nil {
				return err
			}

		case operation.Call:
			vm.frames.current().pointer++ // TODO: this is a hack for temporarily skipping the call
			fn, ok := vm.stack.top().(*data.CompiledFunction)
			if !ok {
				return fmt.Errorf("calling non-function")
			}
			frame := NewFrame(fn, vm.stack.pointer)
			vm.frames.push(frame)
			vm.stack.pointer = frame.framePointer + fn.LocalCount

		case operation.ReturnValue:
			value := vm.stack.pop()
			frame := vm.frames.pop()
			vm.stack.pointer = frame.framePointer - 1
			err := vm.stack.push(value)
			if err != nil {
				return err
			}

		case operation.Return:
			frame := vm.frames.pop()
			vm.stack.pointer = frame.framePointer - 1
			err := vm.stack.push(NULL)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func isTruthy(d data.Data) bool {
	switch d := d.(type) {
	case *data.Boolean:
		return d.Value
	case *data.Null:
		return false
	default:
		return true
	}
}
