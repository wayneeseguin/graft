package operators

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestOperandType(t *testing.T) {
	Convey("OperandType", t, func() {
		Convey("String representation", func() {
			So(TypeInt.String(), ShouldEqual, "int")
			So(TypeFloat.String(), ShouldEqual, "float")
			So(TypeString.String(), ShouldEqual, "string")
			So(TypeBool.String(), ShouldEqual, "bool")
			So(TypeMap.String(), ShouldEqual, "map")
			So(TypeList.String(), ShouldEqual, "list")
			So(TypeNull.String(), ShouldEqual, "null")
			So(TypeUnknown.String(), ShouldEqual, "unknown")
		})
	})
}

func TestGetOperandType(t *testing.T) {
	Convey("GetOperandType", t, func() {
		Convey("identifies integer types", func() {
			So(GetOperandType(42), ShouldEqual, TypeInt)
			So(GetOperandType(int8(42)), ShouldEqual, TypeInt)
			So(GetOperandType(int16(42)), ShouldEqual, TypeInt)
			So(GetOperandType(int32(42)), ShouldEqual, TypeInt)
			So(GetOperandType(int64(42)), ShouldEqual, TypeInt)
			So(GetOperandType(uint(42)), ShouldEqual, TypeInt)
			So(GetOperandType(uint8(42)), ShouldEqual, TypeInt)
			So(GetOperandType(uint16(42)), ShouldEqual, TypeInt)
			So(GetOperandType(uint32(42)), ShouldEqual, TypeInt)
			So(GetOperandType(uint64(42)), ShouldEqual, TypeInt)
		})

		Convey("identifies float types", func() {
			So(GetOperandType(3.14), ShouldEqual, TypeFloat)
			So(GetOperandType(float32(3.14)), ShouldEqual, TypeFloat)
			So(GetOperandType(float64(3.14)), ShouldEqual, TypeFloat)
		})

		Convey("identifies string types", func() {
			So(GetOperandType("hello"), ShouldEqual, TypeString)
			So(GetOperandType(""), ShouldEqual, TypeString)
		})

		Convey("identifies boolean types", func() {
			So(GetOperandType(true), ShouldEqual, TypeBool)
			So(GetOperandType(false), ShouldEqual, TypeBool)
		})

		Convey("identifies map types", func() {
			So(GetOperandType(map[string]interface{}{"key": "value"}), ShouldEqual, TypeMap)
			So(GetOperandType(map[interface{}]interface{}{"key": "value"}), ShouldEqual, TypeMap)
		})

		Convey("identifies list types", func() {
			So(GetOperandType([]interface{}{1, 2, 3}), ShouldEqual, TypeList)
			So(GetOperandType([]interface{}{}), ShouldEqual, TypeList)
		})

		Convey("identifies null types", func() {
			So(GetOperandType(nil), ShouldEqual, TypeNull)
		})

		Convey("identifies unknown types for complex structs", func() {
			type CustomStruct struct{ Field string }
			So(GetOperandType(CustomStruct{Field: "test"}), ShouldEqual, TypeUnknown)
		})
	})
}

func TestTypeRegistry(t *testing.T) {
	Convey("TypeRegistry", t, func() {
		Convey("creates a new registry", func() {
			registry := NewTypeRegistry()
			So(registry, ShouldNotBeNil)
			So(registry.handlers, ShouldNotBeNil)
			So(len(registry.handlers), ShouldEqual, 0)
		})

		Convey("registers handlers sorted by priority", func() {
			registry := NewTypeRegistry()
			
			// Create mock handlers with different priorities
			handler1 := &mockHandler{priority: 10}
			handler2 := &mockHandler{priority: 20}
			handler3 := &mockHandler{priority: 15}
			
			registry.Register(handler1)
			registry.Register(handler2)
			registry.Register(handler3)
			
			So(len(registry.handlers), ShouldEqual, 3)
			// Should be sorted by priority (highest first)
			So(registry.handlers[0], ShouldEqual, handler2) // priority 20
			So(registry.handlers[1], ShouldEqual, handler3) // priority 15
			So(registry.handlers[2], ShouldEqual, handler1) // priority 10
		})

		Convey("finds appropriate handler", func() {
			registry := NewTypeRegistry()
			
			handler := &mockHandler{
				priority: 10,
				canHandleFunc: func(a, b OperandType) bool {
					return a == TypeInt && b == TypeInt
				},
			}
			
			registry.Register(handler)
			
			found := registry.FindHandler(TypeInt, TypeInt)
			So(found, ShouldEqual, handler)
			
			notFound := registry.FindHandler(TypeString, TypeString)
			So(notFound, ShouldBeNil)
		})
	})
}

func TestBaseTypeHandler(t *testing.T) {
	Convey("BaseTypeHandler", t, func() {
		Convey("creates a new base handler", func() {
			handler := NewBaseTypeHandler(10)
			So(handler, ShouldNotBeNil)
			So(handler.Priority(), ShouldEqual, 10)
		})

		Convey("adds supported type pairs", func() {
			handler := NewBaseTypeHandler(10)
			
			handler.AddSupportedTypes(
				TypePair{A: TypeInt, B: TypeInt},
				TypePair{A: TypeInt, B: TypeFloat},
			)
			
			// Should support both directions for each pair
			So(handler.CanHandle(TypeInt, TypeInt), ShouldBeTrue)
			So(handler.CanHandle(TypeInt, TypeFloat), ShouldBeTrue)
			So(handler.CanHandle(TypeFloat, TypeInt), ShouldBeTrue) // Reverse
			
			// Should not support other combinations
			So(handler.CanHandle(TypeString, TypeString), ShouldBeFalse)
			So(handler.CanHandle(TypeInt, TypeString), ShouldBeFalse)
		})
	})
}

func TestNotImplementedError(t *testing.T) {
	Convey("NotImplementedError", t, func() {
		err := NotImplementedError("add", 42, "hello")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "add operation not supported")
		So(err.Error(), ShouldContainSubstring, "int")
		So(err.Error(), ShouldContainSubstring, "string")
	})
}

func TestGetGlobalTypeRegistry(t *testing.T) {
	Convey("GetGlobalTypeRegistry", t, func() {
		registry := GetGlobalTypeRegistry()
		So(registry, ShouldNotBeNil)
		So(registry, ShouldEqual, globalTypeRegistry)
	})
}

// Mock handler for testing
type mockHandler struct {
	priority      int
	canHandleFunc func(a, b OperandType) bool
}

func (h *mockHandler) Add(a, b interface{}) (interface{}, error) {
	return nil, NotImplementedError("add", a, b)
}

func (h *mockHandler) Subtract(a, b interface{}) (interface{}, error) {
	return nil, NotImplementedError("subtract", a, b)
}

func (h *mockHandler) Multiply(a, b interface{}) (interface{}, error) {
	return nil, NotImplementedError("multiply", a, b)
}

func (h *mockHandler) Divide(a, b interface{}) (interface{}, error) {
	return nil, NotImplementedError("divide", a, b)
}

func (h *mockHandler) Modulo(a, b interface{}) (interface{}, error) {
	return nil, NotImplementedError("modulo", a, b)
}

func (h *mockHandler) Equal(a, b interface{}) (bool, error) {
	return false, NotImplementedError("equal", a, b)
}

func (h *mockHandler) NotEqual(a, b interface{}) (bool, error) {
	return false, NotImplementedError("not equal", a, b)
}

func (h *mockHandler) Less(a, b interface{}) (bool, error) {
	return false, NotImplementedError("less", a, b)
}

func (h *mockHandler) Greater(a, b interface{}) (bool, error) {
	return false, NotImplementedError("greater", a, b)
}

func (h *mockHandler) LessOrEqual(a, b interface{}) (bool, error) {
	return false, NotImplementedError("less or equal", a, b)
}

func (h *mockHandler) GreaterOrEqual(a, b interface{}) (bool, error) {
	return false, NotImplementedError("greater or equal", a, b)
}

func (h *mockHandler) CanHandle(aType, bType OperandType) bool {
	if h.canHandleFunc != nil {
		return h.canHandleFunc(aType, bType)
	}
	return false
}

func (h *mockHandler) Priority() int {
	return h.priority
}