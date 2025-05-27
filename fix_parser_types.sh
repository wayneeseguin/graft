#!/bin/bash

echo "Fixing parser type references..."

# Update all parser files to use graft. prefix for types
for file in pkg/graft/parser/*.go; do
    [ -f "$file" ] || continue
    
    echo "Updating $file..."
    
    # Update type references to use graft. prefix
    sed -i '' \
        -e 's/\bOperatorPhase\b/graft.OperatorPhase/g' \
        -e 's/\bOpcall\b/graft.Opcall/g' \
        -e 's/\bExpr\b/graft.Expr/g' \
        -e 's/\bExprType\b/graft.ExprType/g' \
        -e 's/\bOperator\b/graft.Operator/g' \
        -e 's/\bEvaluator\b/graft.Evaluator/g' \
        -e 's/\bResponse\b/graft.Response/g' \
        -e 's/\bAction\b/graft.Action/g' \
        -e 's/\bLiteral\b/graft.Literal/g' \
        -e 's/\bReference\b/graft.Reference/g' \
        -e 's/\bOperatorCall\b/graft.OperatorCall/g' \
        -e 's/\bLogicalOr\b/graft.LogicalOr/g' \
        -e 's/\bEnvVar\b/graft.EnvVar/g' \
        -e 's/\bPosition\b/graft.Position/g' \
        -e 's/\bWarningError\b/graft.WarningError/g' \
        -e 's/\bMultiError\b/graft.MultiError/g' \
        -e 's/\bErrorContext\b/graft.ErrorContext/g' \
        -e 's/\beContextAll\b/graft.eContextAll/g' \
        -e 's/\bDEBUG\b/log.DEBUG/g' \
        -e 's/\bTRACE\b/log.TRACE/g' \
        "$file"
    
    # Fix double prefixes (e.g., graft.graft.OperatorPhase)
    sed -i '' 's/graft\.graft\./graft./g' "$file"
    
    # Fix function signatures that shouldn't have prefixes
    sed -i '' \
        -e 's/func ParseOpcallCompat(phase graft\./func ParseOpcallCompat(phase graft./g' \
        -e 's/func (\*graft\./func (*/g' \
        "$file"
done

# Also need to fix global variables
for file in pkg/graft/parser/*.go; do
    [ -f "$file" ] || continue
    
    # Update global variable references
    sed -i '' \
        -e 's/\bUseEnhancedParser\b/graft.UseEnhancedParser/g' \
        -e 's/\bOperatorFor\b/operators.OperatorFor/g' \
        -e 's/\bRegisterOp\b/operators.RegisterOp/g' \
        -e 's/\bParseOpcall\b/graft.ParseOpcall/g' \
        -e 's/\bNewWarningError\b/graft.NewWarningError/g' \
        -e 's/\bResolveOperatorArgument\b/operators.ResolveOperatorArgument/g' \
        "$file"
done

echo "Parser type references updated!"