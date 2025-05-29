package operators

// init registers all type handlers with the global type registry
func init() {
	// Register type handlers in priority order
	globalTypeRegistry.Register(NewNumericTypeHandler())
	globalTypeRegistry.Register(NewStringTypeHandler())
	globalTypeRegistry.Register(NewBooleanTypeHandler())
	
	// TODO: Register additional handlers as they are implemented
	// globalTypeRegistry.Register(NewMapTypeHandler())
	// globalTypeRegistry.Register(NewListTypeHandler())
	// globalTypeRegistry.Register(NewMixedTypeHandler())
}