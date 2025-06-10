package graft

// GetEngine returns the engine from an evaluator
func GetEngine(ev *Evaluator) Engine {
	if ev.engine != nil {
		if eng, ok := ev.engine.(Engine); ok {
			return eng
		}
	}
	// Return a default engine for backward compatibility
	engine, _ := CreateDefaultEngine()
	return engine
}
