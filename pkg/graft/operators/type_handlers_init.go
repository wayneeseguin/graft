package operators

// init registers all type handlers with the global type registry
func init() {
	// Register type handlers in priority order (highest priority first)
	globalTypeRegistry.Register(NewNumericTypeHandler())     // Priority 50
	globalTypeRegistry.Register(NewStringTypeHandler())      // Priority 50  
	globalTypeRegistry.Register(NewBooleanTypeHandler())     // Priority 50
	globalTypeRegistry.Register(NewMapTypeHandler())         // Priority 70
	globalTypeRegistry.Register(NewListTypeHandler())        // Priority 70
	
	// Register mixed type handler last (lowest priority, fallback)
	globalTypeRegistry.Register(NewMixedTypeHandler())       // Priority 10
}