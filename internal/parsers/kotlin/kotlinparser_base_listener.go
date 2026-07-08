// Code generated from ./grammars/kotlin/KotlinParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package kotlin // KotlinParser
import "github.com/antlr4-go/antlr/v4"

// BaseKotlinParserListener is a complete listener for a parse tree produced by KotlinParser.
type BaseKotlinParserListener struct{}

var _ KotlinParserListener = &BaseKotlinParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseKotlinParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseKotlinParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseKotlinParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseKotlinParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterKotlinFile is called when production kotlinFile is entered.
func (s *BaseKotlinParserListener) EnterKotlinFile(ctx *KotlinFileContext) {}

// ExitKotlinFile is called when production kotlinFile is exited.
func (s *BaseKotlinParserListener) ExitKotlinFile(ctx *KotlinFileContext) {}

// EnterScript is called when production script is entered.
func (s *BaseKotlinParserListener) EnterScript(ctx *ScriptContext) {}

// ExitScript is called when production script is exited.
func (s *BaseKotlinParserListener) ExitScript(ctx *ScriptContext) {}

// EnterPreamble is called when production preamble is entered.
func (s *BaseKotlinParserListener) EnterPreamble(ctx *PreambleContext) {}

// ExitPreamble is called when production preamble is exited.
func (s *BaseKotlinParserListener) ExitPreamble(ctx *PreambleContext) {}

// EnterFileAnnotations is called when production fileAnnotations is entered.
func (s *BaseKotlinParserListener) EnterFileAnnotations(ctx *FileAnnotationsContext) {}

// ExitFileAnnotations is called when production fileAnnotations is exited.
func (s *BaseKotlinParserListener) ExitFileAnnotations(ctx *FileAnnotationsContext) {}

// EnterFileAnnotation is called when production fileAnnotation is entered.
func (s *BaseKotlinParserListener) EnterFileAnnotation(ctx *FileAnnotationContext) {}

// ExitFileAnnotation is called when production fileAnnotation is exited.
func (s *BaseKotlinParserListener) ExitFileAnnotation(ctx *FileAnnotationContext) {}

// EnterPackageHeader is called when production packageHeader is entered.
func (s *BaseKotlinParserListener) EnterPackageHeader(ctx *PackageHeaderContext) {}

// ExitPackageHeader is called when production packageHeader is exited.
func (s *BaseKotlinParserListener) ExitPackageHeader(ctx *PackageHeaderContext) {}

// EnterImportList is called when production importList is entered.
func (s *BaseKotlinParserListener) EnterImportList(ctx *ImportListContext) {}

// ExitImportList is called when production importList is exited.
func (s *BaseKotlinParserListener) ExitImportList(ctx *ImportListContext) {}

// EnterImportHeader is called when production importHeader is entered.
func (s *BaseKotlinParserListener) EnterImportHeader(ctx *ImportHeaderContext) {}

// ExitImportHeader is called when production importHeader is exited.
func (s *BaseKotlinParserListener) ExitImportHeader(ctx *ImportHeaderContext) {}

// EnterImportAlias is called when production importAlias is entered.
func (s *BaseKotlinParserListener) EnterImportAlias(ctx *ImportAliasContext) {}

// ExitImportAlias is called when production importAlias is exited.
func (s *BaseKotlinParserListener) ExitImportAlias(ctx *ImportAliasContext) {}

// EnterTopLevelObject is called when production topLevelObject is entered.
func (s *BaseKotlinParserListener) EnterTopLevelObject(ctx *TopLevelObjectContext) {}

// ExitTopLevelObject is called when production topLevelObject is exited.
func (s *BaseKotlinParserListener) ExitTopLevelObject(ctx *TopLevelObjectContext) {}

// EnterClassDeclaration is called when production classDeclaration is entered.
func (s *BaseKotlinParserListener) EnterClassDeclaration(ctx *ClassDeclarationContext) {}

// ExitClassDeclaration is called when production classDeclaration is exited.
func (s *BaseKotlinParserListener) ExitClassDeclaration(ctx *ClassDeclarationContext) {}

// EnterPrimaryConstructor is called when production primaryConstructor is entered.
func (s *BaseKotlinParserListener) EnterPrimaryConstructor(ctx *PrimaryConstructorContext) {}

// ExitPrimaryConstructor is called when production primaryConstructor is exited.
func (s *BaseKotlinParserListener) ExitPrimaryConstructor(ctx *PrimaryConstructorContext) {}

// EnterClassParameters is called when production classParameters is entered.
func (s *BaseKotlinParserListener) EnterClassParameters(ctx *ClassParametersContext) {}

// ExitClassParameters is called when production classParameters is exited.
func (s *BaseKotlinParserListener) ExitClassParameters(ctx *ClassParametersContext) {}

// EnterClassParameter is called when production classParameter is entered.
func (s *BaseKotlinParserListener) EnterClassParameter(ctx *ClassParameterContext) {}

// ExitClassParameter is called when production classParameter is exited.
func (s *BaseKotlinParserListener) ExitClassParameter(ctx *ClassParameterContext) {}

// EnterDelegationSpecifiers is called when production delegationSpecifiers is entered.
func (s *BaseKotlinParserListener) EnterDelegationSpecifiers(ctx *DelegationSpecifiersContext) {}

// ExitDelegationSpecifiers is called when production delegationSpecifiers is exited.
func (s *BaseKotlinParserListener) ExitDelegationSpecifiers(ctx *DelegationSpecifiersContext) {}

// EnterDelegationSpecifier is called when production delegationSpecifier is entered.
func (s *BaseKotlinParserListener) EnterDelegationSpecifier(ctx *DelegationSpecifierContext) {}

// ExitDelegationSpecifier is called when production delegationSpecifier is exited.
func (s *BaseKotlinParserListener) ExitDelegationSpecifier(ctx *DelegationSpecifierContext) {}

// EnterConstructorInvocation is called when production constructorInvocation is entered.
func (s *BaseKotlinParserListener) EnterConstructorInvocation(ctx *ConstructorInvocationContext) {}

// ExitConstructorInvocation is called when production constructorInvocation is exited.
func (s *BaseKotlinParserListener) ExitConstructorInvocation(ctx *ConstructorInvocationContext) {}

// EnterExplicitDelegation is called when production explicitDelegation is entered.
func (s *BaseKotlinParserListener) EnterExplicitDelegation(ctx *ExplicitDelegationContext) {}

// ExitExplicitDelegation is called when production explicitDelegation is exited.
func (s *BaseKotlinParserListener) ExitExplicitDelegation(ctx *ExplicitDelegationContext) {}

// EnterClassBody is called when production classBody is entered.
func (s *BaseKotlinParserListener) EnterClassBody(ctx *ClassBodyContext) {}

// ExitClassBody is called when production classBody is exited.
func (s *BaseKotlinParserListener) ExitClassBody(ctx *ClassBodyContext) {}

// EnterClassMemberDeclaration is called when production classMemberDeclaration is entered.
func (s *BaseKotlinParserListener) EnterClassMemberDeclaration(ctx *ClassMemberDeclarationContext) {}

// ExitClassMemberDeclaration is called when production classMemberDeclaration is exited.
func (s *BaseKotlinParserListener) ExitClassMemberDeclaration(ctx *ClassMemberDeclarationContext) {}

// EnterAnonymousInitializer is called when production anonymousInitializer is entered.
func (s *BaseKotlinParserListener) EnterAnonymousInitializer(ctx *AnonymousInitializerContext) {}

// ExitAnonymousInitializer is called when production anonymousInitializer is exited.
func (s *BaseKotlinParserListener) ExitAnonymousInitializer(ctx *AnonymousInitializerContext) {}

// EnterSecondaryConstructor is called when production secondaryConstructor is entered.
func (s *BaseKotlinParserListener) EnterSecondaryConstructor(ctx *SecondaryConstructorContext) {}

// ExitSecondaryConstructor is called when production secondaryConstructor is exited.
func (s *BaseKotlinParserListener) ExitSecondaryConstructor(ctx *SecondaryConstructorContext) {}

// EnterConstructorDelegationCall is called when production constructorDelegationCall is entered.
func (s *BaseKotlinParserListener) EnterConstructorDelegationCall(ctx *ConstructorDelegationCallContext) {
}

// ExitConstructorDelegationCall is called when production constructorDelegationCall is exited.
func (s *BaseKotlinParserListener) ExitConstructorDelegationCall(ctx *ConstructorDelegationCallContext) {
}

// EnterEnumClassBody is called when production enumClassBody is entered.
func (s *BaseKotlinParserListener) EnterEnumClassBody(ctx *EnumClassBodyContext) {}

// ExitEnumClassBody is called when production enumClassBody is exited.
func (s *BaseKotlinParserListener) ExitEnumClassBody(ctx *EnumClassBodyContext) {}

// EnterEnumEntries is called when production enumEntries is entered.
func (s *BaseKotlinParserListener) EnterEnumEntries(ctx *EnumEntriesContext) {}

// ExitEnumEntries is called when production enumEntries is exited.
func (s *BaseKotlinParserListener) ExitEnumEntries(ctx *EnumEntriesContext) {}

// EnterEnumEntry is called when production enumEntry is entered.
func (s *BaseKotlinParserListener) EnterEnumEntry(ctx *EnumEntryContext) {}

// ExitEnumEntry is called when production enumEntry is exited.
func (s *BaseKotlinParserListener) ExitEnumEntry(ctx *EnumEntryContext) {}

// EnterFunctionDeclaration is called when production functionDeclaration is entered.
func (s *BaseKotlinParserListener) EnterFunctionDeclaration(ctx *FunctionDeclarationContext) {}

// ExitFunctionDeclaration is called when production functionDeclaration is exited.
func (s *BaseKotlinParserListener) ExitFunctionDeclaration(ctx *FunctionDeclarationContext) {}

// EnterFunctionValueParameters is called when production functionValueParameters is entered.
func (s *BaseKotlinParserListener) EnterFunctionValueParameters(ctx *FunctionValueParametersContext) {
}

// ExitFunctionValueParameters is called when production functionValueParameters is exited.
func (s *BaseKotlinParserListener) ExitFunctionValueParameters(ctx *FunctionValueParametersContext) {}

// EnterFunctionValueParameter is called when production functionValueParameter is entered.
func (s *BaseKotlinParserListener) EnterFunctionValueParameter(ctx *FunctionValueParameterContext) {}

// ExitFunctionValueParameter is called when production functionValueParameter is exited.
func (s *BaseKotlinParserListener) ExitFunctionValueParameter(ctx *FunctionValueParameterContext) {}

// EnterParameter is called when production parameter is entered.
func (s *BaseKotlinParserListener) EnterParameter(ctx *ParameterContext) {}

// ExitParameter is called when production parameter is exited.
func (s *BaseKotlinParserListener) ExitParameter(ctx *ParameterContext) {}

// EnterReceiverType is called when production receiverType is entered.
func (s *BaseKotlinParserListener) EnterReceiverType(ctx *ReceiverTypeContext) {}

// ExitReceiverType is called when production receiverType is exited.
func (s *BaseKotlinParserListener) ExitReceiverType(ctx *ReceiverTypeContext) {}

// EnterFunctionBody is called when production functionBody is entered.
func (s *BaseKotlinParserListener) EnterFunctionBody(ctx *FunctionBodyContext) {}

// ExitFunctionBody is called when production functionBody is exited.
func (s *BaseKotlinParserListener) ExitFunctionBody(ctx *FunctionBodyContext) {}

// EnterObjectDeclaration is called when production objectDeclaration is entered.
func (s *BaseKotlinParserListener) EnterObjectDeclaration(ctx *ObjectDeclarationContext) {}

// ExitObjectDeclaration is called when production objectDeclaration is exited.
func (s *BaseKotlinParserListener) ExitObjectDeclaration(ctx *ObjectDeclarationContext) {}

// EnterCompanionObject is called when production companionObject is entered.
func (s *BaseKotlinParserListener) EnterCompanionObject(ctx *CompanionObjectContext) {}

// ExitCompanionObject is called when production companionObject is exited.
func (s *BaseKotlinParserListener) ExitCompanionObject(ctx *CompanionObjectContext) {}

// EnterPropertyDeclaration is called when production propertyDeclaration is entered.
func (s *BaseKotlinParserListener) EnterPropertyDeclaration(ctx *PropertyDeclarationContext) {}

// ExitPropertyDeclaration is called when production propertyDeclaration is exited.
func (s *BaseKotlinParserListener) ExitPropertyDeclaration(ctx *PropertyDeclarationContext) {}

// EnterMultiVariableDeclaration is called when production multiVariableDeclaration is entered.
func (s *BaseKotlinParserListener) EnterMultiVariableDeclaration(ctx *MultiVariableDeclarationContext) {
}

// ExitMultiVariableDeclaration is called when production multiVariableDeclaration is exited.
func (s *BaseKotlinParserListener) ExitMultiVariableDeclaration(ctx *MultiVariableDeclarationContext) {
}

// EnterVariableDeclaration is called when production variableDeclaration is entered.
func (s *BaseKotlinParserListener) EnterVariableDeclaration(ctx *VariableDeclarationContext) {}

// ExitVariableDeclaration is called when production variableDeclaration is exited.
func (s *BaseKotlinParserListener) ExitVariableDeclaration(ctx *VariableDeclarationContext) {}

// EnterGetter is called when production getter is entered.
func (s *BaseKotlinParserListener) EnterGetter(ctx *GetterContext) {}

// ExitGetter is called when production getter is exited.
func (s *BaseKotlinParserListener) ExitGetter(ctx *GetterContext) {}

// EnterSetter is called when production setter is entered.
func (s *BaseKotlinParserListener) EnterSetter(ctx *SetterContext) {}

// ExitSetter is called when production setter is exited.
func (s *BaseKotlinParserListener) ExitSetter(ctx *SetterContext) {}

// EnterTypeAlias is called when production typeAlias is entered.
func (s *BaseKotlinParserListener) EnterTypeAlias(ctx *TypeAliasContext) {}

// ExitTypeAlias is called when production typeAlias is exited.
func (s *BaseKotlinParserListener) ExitTypeAlias(ctx *TypeAliasContext) {}

// EnterTypeParameters is called when production typeParameters is entered.
func (s *BaseKotlinParserListener) EnterTypeParameters(ctx *TypeParametersContext) {}

// ExitTypeParameters is called when production typeParameters is exited.
func (s *BaseKotlinParserListener) ExitTypeParameters(ctx *TypeParametersContext) {}

// EnterTypeParameter is called when production typeParameter is entered.
func (s *BaseKotlinParserListener) EnterTypeParameter(ctx *TypeParameterContext) {}

// ExitTypeParameter is called when production typeParameter is exited.
func (s *BaseKotlinParserListener) ExitTypeParameter(ctx *TypeParameterContext) {}

// EnterType is called when production type is entered.
func (s *BaseKotlinParserListener) EnterType(ctx *TypeContext) {}

// ExitType is called when production type is exited.
func (s *BaseKotlinParserListener) ExitType(ctx *TypeContext) {}

// EnterTypeModifierList is called when production typeModifierList is entered.
func (s *BaseKotlinParserListener) EnterTypeModifierList(ctx *TypeModifierListContext) {}

// ExitTypeModifierList is called when production typeModifierList is exited.
func (s *BaseKotlinParserListener) ExitTypeModifierList(ctx *TypeModifierListContext) {}

// EnterParenthesizedType is called when production parenthesizedType is entered.
func (s *BaseKotlinParserListener) EnterParenthesizedType(ctx *ParenthesizedTypeContext) {}

// ExitParenthesizedType is called when production parenthesizedType is exited.
func (s *BaseKotlinParserListener) ExitParenthesizedType(ctx *ParenthesizedTypeContext) {}

// EnterNullableType is called when production nullableType is entered.
func (s *BaseKotlinParserListener) EnterNullableType(ctx *NullableTypeContext) {}

// ExitNullableType is called when production nullableType is exited.
func (s *BaseKotlinParserListener) ExitNullableType(ctx *NullableTypeContext) {}

// EnterTypeReference is called when production typeReference is entered.
func (s *BaseKotlinParserListener) EnterTypeReference(ctx *TypeReferenceContext) {}

// ExitTypeReference is called when production typeReference is exited.
func (s *BaseKotlinParserListener) ExitTypeReference(ctx *TypeReferenceContext) {}

// EnterFunctionType is called when production functionType is entered.
func (s *BaseKotlinParserListener) EnterFunctionType(ctx *FunctionTypeContext) {}

// ExitFunctionType is called when production functionType is exited.
func (s *BaseKotlinParserListener) ExitFunctionType(ctx *FunctionTypeContext) {}

// EnterFunctionTypeReceiver is called when production functionTypeReceiver is entered.
func (s *BaseKotlinParserListener) EnterFunctionTypeReceiver(ctx *FunctionTypeReceiverContext) {}

// ExitFunctionTypeReceiver is called when production functionTypeReceiver is exited.
func (s *BaseKotlinParserListener) ExitFunctionTypeReceiver(ctx *FunctionTypeReceiverContext) {}

// EnterUserType is called when production userType is entered.
func (s *BaseKotlinParserListener) EnterUserType(ctx *UserTypeContext) {}

// ExitUserType is called when production userType is exited.
func (s *BaseKotlinParserListener) ExitUserType(ctx *UserTypeContext) {}

// EnterSimpleUserType is called when production simpleUserType is entered.
func (s *BaseKotlinParserListener) EnterSimpleUserType(ctx *SimpleUserTypeContext) {}

// ExitSimpleUserType is called when production simpleUserType is exited.
func (s *BaseKotlinParserListener) ExitSimpleUserType(ctx *SimpleUserTypeContext) {}

// EnterFunctionTypeParameters is called when production functionTypeParameters is entered.
func (s *BaseKotlinParserListener) EnterFunctionTypeParameters(ctx *FunctionTypeParametersContext) {}

// ExitFunctionTypeParameters is called when production functionTypeParameters is exited.
func (s *BaseKotlinParserListener) ExitFunctionTypeParameters(ctx *FunctionTypeParametersContext) {}

// EnterTypeConstraints is called when production typeConstraints is entered.
func (s *BaseKotlinParserListener) EnterTypeConstraints(ctx *TypeConstraintsContext) {}

// ExitTypeConstraints is called when production typeConstraints is exited.
func (s *BaseKotlinParserListener) ExitTypeConstraints(ctx *TypeConstraintsContext) {}

// EnterTypeConstraint is called when production typeConstraint is entered.
func (s *BaseKotlinParserListener) EnterTypeConstraint(ctx *TypeConstraintContext) {}

// ExitTypeConstraint is called when production typeConstraint is exited.
func (s *BaseKotlinParserListener) ExitTypeConstraint(ctx *TypeConstraintContext) {}

// EnterBlock is called when production block is entered.
func (s *BaseKotlinParserListener) EnterBlock(ctx *BlockContext) {}

// ExitBlock is called when production block is exited.
func (s *BaseKotlinParserListener) ExitBlock(ctx *BlockContext) {}

// EnterStatements is called when production statements is entered.
func (s *BaseKotlinParserListener) EnterStatements(ctx *StatementsContext) {}

// ExitStatements is called when production statements is exited.
func (s *BaseKotlinParserListener) ExitStatements(ctx *StatementsContext) {}

// EnterStatement is called when production statement is entered.
func (s *BaseKotlinParserListener) EnterStatement(ctx *StatementContext) {}

// ExitStatement is called when production statement is exited.
func (s *BaseKotlinParserListener) ExitStatement(ctx *StatementContext) {}

// EnterBlockLevelExpression is called when production blockLevelExpression is entered.
func (s *BaseKotlinParserListener) EnterBlockLevelExpression(ctx *BlockLevelExpressionContext) {}

// ExitBlockLevelExpression is called when production blockLevelExpression is exited.
func (s *BaseKotlinParserListener) ExitBlockLevelExpression(ctx *BlockLevelExpressionContext) {}

// EnterDeclaration is called when production declaration is entered.
func (s *BaseKotlinParserListener) EnterDeclaration(ctx *DeclarationContext) {}

// ExitDeclaration is called when production declaration is exited.
func (s *BaseKotlinParserListener) ExitDeclaration(ctx *DeclarationContext) {}

// EnterExpression is called when production expression is entered.
func (s *BaseKotlinParserListener) EnterExpression(ctx *ExpressionContext) {}

// ExitExpression is called when production expression is exited.
func (s *BaseKotlinParserListener) ExitExpression(ctx *ExpressionContext) {}

// EnterDisjunction is called when production disjunction is entered.
func (s *BaseKotlinParserListener) EnterDisjunction(ctx *DisjunctionContext) {}

// ExitDisjunction is called when production disjunction is exited.
func (s *BaseKotlinParserListener) ExitDisjunction(ctx *DisjunctionContext) {}

// EnterConjunction is called when production conjunction is entered.
func (s *BaseKotlinParserListener) EnterConjunction(ctx *ConjunctionContext) {}

// ExitConjunction is called when production conjunction is exited.
func (s *BaseKotlinParserListener) ExitConjunction(ctx *ConjunctionContext) {}

// EnterEqualityComparison is called when production equalityComparison is entered.
func (s *BaseKotlinParserListener) EnterEqualityComparison(ctx *EqualityComparisonContext) {}

// ExitEqualityComparison is called when production equalityComparison is exited.
func (s *BaseKotlinParserListener) ExitEqualityComparison(ctx *EqualityComparisonContext) {}

// EnterComparison is called when production comparison is entered.
func (s *BaseKotlinParserListener) EnterComparison(ctx *ComparisonContext) {}

// ExitComparison is called when production comparison is exited.
func (s *BaseKotlinParserListener) ExitComparison(ctx *ComparisonContext) {}

// EnterNamedInfix is called when production namedInfix is entered.
func (s *BaseKotlinParserListener) EnterNamedInfix(ctx *NamedInfixContext) {}

// ExitNamedInfix is called when production namedInfix is exited.
func (s *BaseKotlinParserListener) ExitNamedInfix(ctx *NamedInfixContext) {}

// EnterElvisExpression is called when production elvisExpression is entered.
func (s *BaseKotlinParserListener) EnterElvisExpression(ctx *ElvisExpressionContext) {}

// ExitElvisExpression is called when production elvisExpression is exited.
func (s *BaseKotlinParserListener) ExitElvisExpression(ctx *ElvisExpressionContext) {}

// EnterInfixFunctionCall is called when production infixFunctionCall is entered.
func (s *BaseKotlinParserListener) EnterInfixFunctionCall(ctx *InfixFunctionCallContext) {}

// ExitInfixFunctionCall is called when production infixFunctionCall is exited.
func (s *BaseKotlinParserListener) ExitInfixFunctionCall(ctx *InfixFunctionCallContext) {}

// EnterRangeExpression is called when production rangeExpression is entered.
func (s *BaseKotlinParserListener) EnterRangeExpression(ctx *RangeExpressionContext) {}

// ExitRangeExpression is called when production rangeExpression is exited.
func (s *BaseKotlinParserListener) ExitRangeExpression(ctx *RangeExpressionContext) {}

// EnterAdditiveExpression is called when production additiveExpression is entered.
func (s *BaseKotlinParserListener) EnterAdditiveExpression(ctx *AdditiveExpressionContext) {}

// ExitAdditiveExpression is called when production additiveExpression is exited.
func (s *BaseKotlinParserListener) ExitAdditiveExpression(ctx *AdditiveExpressionContext) {}

// EnterMultiplicativeExpression is called when production multiplicativeExpression is entered.
func (s *BaseKotlinParserListener) EnterMultiplicativeExpression(ctx *MultiplicativeExpressionContext) {
}

// ExitMultiplicativeExpression is called when production multiplicativeExpression is exited.
func (s *BaseKotlinParserListener) ExitMultiplicativeExpression(ctx *MultiplicativeExpressionContext) {
}

// EnterTypeRHS is called when production typeRHS is entered.
func (s *BaseKotlinParserListener) EnterTypeRHS(ctx *TypeRHSContext) {}

// ExitTypeRHS is called when production typeRHS is exited.
func (s *BaseKotlinParserListener) ExitTypeRHS(ctx *TypeRHSContext) {}

// EnterPrefixUnaryExpression is called when production prefixUnaryExpression is entered.
func (s *BaseKotlinParserListener) EnterPrefixUnaryExpression(ctx *PrefixUnaryExpressionContext) {}

// ExitPrefixUnaryExpression is called when production prefixUnaryExpression is exited.
func (s *BaseKotlinParserListener) ExitPrefixUnaryExpression(ctx *PrefixUnaryExpressionContext) {}

// EnterPostfixUnaryExpression is called when production postfixUnaryExpression is entered.
func (s *BaseKotlinParserListener) EnterPostfixUnaryExpression(ctx *PostfixUnaryExpressionContext) {}

// ExitPostfixUnaryExpression is called when production postfixUnaryExpression is exited.
func (s *BaseKotlinParserListener) ExitPostfixUnaryExpression(ctx *PostfixUnaryExpressionContext) {}

// EnterAtomicExpression is called when production atomicExpression is entered.
func (s *BaseKotlinParserListener) EnterAtomicExpression(ctx *AtomicExpressionContext) {}

// ExitAtomicExpression is called when production atomicExpression is exited.
func (s *BaseKotlinParserListener) ExitAtomicExpression(ctx *AtomicExpressionContext) {}

// EnterParenthesizedExpression is called when production parenthesizedExpression is entered.
func (s *BaseKotlinParserListener) EnterParenthesizedExpression(ctx *ParenthesizedExpressionContext) {
}

// ExitParenthesizedExpression is called when production parenthesizedExpression is exited.
func (s *BaseKotlinParserListener) ExitParenthesizedExpression(ctx *ParenthesizedExpressionContext) {}

// EnterCallSuffix is called when production callSuffix is entered.
func (s *BaseKotlinParserListener) EnterCallSuffix(ctx *CallSuffixContext) {}

// ExitCallSuffix is called when production callSuffix is exited.
func (s *BaseKotlinParserListener) ExitCallSuffix(ctx *CallSuffixContext) {}

// EnterAnnotatedLambda is called when production annotatedLambda is entered.
func (s *BaseKotlinParserListener) EnterAnnotatedLambda(ctx *AnnotatedLambdaContext) {}

// ExitAnnotatedLambda is called when production annotatedLambda is exited.
func (s *BaseKotlinParserListener) ExitAnnotatedLambda(ctx *AnnotatedLambdaContext) {}

// EnterArrayAccess is called when production arrayAccess is entered.
func (s *BaseKotlinParserListener) EnterArrayAccess(ctx *ArrayAccessContext) {}

// ExitArrayAccess is called when production arrayAccess is exited.
func (s *BaseKotlinParserListener) ExitArrayAccess(ctx *ArrayAccessContext) {}

// EnterValueArguments is called when production valueArguments is entered.
func (s *BaseKotlinParserListener) EnterValueArguments(ctx *ValueArgumentsContext) {}

// ExitValueArguments is called when production valueArguments is exited.
func (s *BaseKotlinParserListener) ExitValueArguments(ctx *ValueArgumentsContext) {}

// EnterTypeArguments is called when production typeArguments is entered.
func (s *BaseKotlinParserListener) EnterTypeArguments(ctx *TypeArgumentsContext) {}

// ExitTypeArguments is called when production typeArguments is exited.
func (s *BaseKotlinParserListener) ExitTypeArguments(ctx *TypeArgumentsContext) {}

// EnterTypeProjection is called when production typeProjection is entered.
func (s *BaseKotlinParserListener) EnterTypeProjection(ctx *TypeProjectionContext) {}

// ExitTypeProjection is called when production typeProjection is exited.
func (s *BaseKotlinParserListener) ExitTypeProjection(ctx *TypeProjectionContext) {}

// EnterTypeProjectionModifierList is called when production typeProjectionModifierList is entered.
func (s *BaseKotlinParserListener) EnterTypeProjectionModifierList(ctx *TypeProjectionModifierListContext) {
}

// ExitTypeProjectionModifierList is called when production typeProjectionModifierList is exited.
func (s *BaseKotlinParserListener) ExitTypeProjectionModifierList(ctx *TypeProjectionModifierListContext) {
}

// EnterValueArgument is called when production valueArgument is entered.
func (s *BaseKotlinParserListener) EnterValueArgument(ctx *ValueArgumentContext) {}

// ExitValueArgument is called when production valueArgument is exited.
func (s *BaseKotlinParserListener) ExitValueArgument(ctx *ValueArgumentContext) {}

// EnterLiteralConstant is called when production literalConstant is entered.
func (s *BaseKotlinParserListener) EnterLiteralConstant(ctx *LiteralConstantContext) {}

// ExitLiteralConstant is called when production literalConstant is exited.
func (s *BaseKotlinParserListener) ExitLiteralConstant(ctx *LiteralConstantContext) {}

// EnterStringLiteral is called when production stringLiteral is entered.
func (s *BaseKotlinParserListener) EnterStringLiteral(ctx *StringLiteralContext) {}

// ExitStringLiteral is called when production stringLiteral is exited.
func (s *BaseKotlinParserListener) ExitStringLiteral(ctx *StringLiteralContext) {}

// EnterLineStringLiteral is called when production lineStringLiteral is entered.
func (s *BaseKotlinParserListener) EnterLineStringLiteral(ctx *LineStringLiteralContext) {}

// ExitLineStringLiteral is called when production lineStringLiteral is exited.
func (s *BaseKotlinParserListener) ExitLineStringLiteral(ctx *LineStringLiteralContext) {}

// EnterMultiLineStringLiteral is called when production multiLineStringLiteral is entered.
func (s *BaseKotlinParserListener) EnterMultiLineStringLiteral(ctx *MultiLineStringLiteralContext) {}

// ExitMultiLineStringLiteral is called when production multiLineStringLiteral is exited.
func (s *BaseKotlinParserListener) ExitMultiLineStringLiteral(ctx *MultiLineStringLiteralContext) {}

// EnterLineStringContent is called when production lineStringContent is entered.
func (s *BaseKotlinParserListener) EnterLineStringContent(ctx *LineStringContentContext) {}

// ExitLineStringContent is called when production lineStringContent is exited.
func (s *BaseKotlinParserListener) ExitLineStringContent(ctx *LineStringContentContext) {}

// EnterLineStringExpression is called when production lineStringExpression is entered.
func (s *BaseKotlinParserListener) EnterLineStringExpression(ctx *LineStringExpressionContext) {}

// ExitLineStringExpression is called when production lineStringExpression is exited.
func (s *BaseKotlinParserListener) ExitLineStringExpression(ctx *LineStringExpressionContext) {}

// EnterMultiLineStringContent is called when production multiLineStringContent is entered.
func (s *BaseKotlinParserListener) EnterMultiLineStringContent(ctx *MultiLineStringContentContext) {}

// ExitMultiLineStringContent is called when production multiLineStringContent is exited.
func (s *BaseKotlinParserListener) ExitMultiLineStringContent(ctx *MultiLineStringContentContext) {}

// EnterMultiLineStringExpression is called when production multiLineStringExpression is entered.
func (s *BaseKotlinParserListener) EnterMultiLineStringExpression(ctx *MultiLineStringExpressionContext) {
}

// ExitMultiLineStringExpression is called when production multiLineStringExpression is exited.
func (s *BaseKotlinParserListener) ExitMultiLineStringExpression(ctx *MultiLineStringExpressionContext) {
}

// EnterFunctionLiteral is called when production functionLiteral is entered.
func (s *BaseKotlinParserListener) EnterFunctionLiteral(ctx *FunctionLiteralContext) {}

// ExitFunctionLiteral is called when production functionLiteral is exited.
func (s *BaseKotlinParserListener) ExitFunctionLiteral(ctx *FunctionLiteralContext) {}

// EnterLambdaParameters is called when production lambdaParameters is entered.
func (s *BaseKotlinParserListener) EnterLambdaParameters(ctx *LambdaParametersContext) {}

// ExitLambdaParameters is called when production lambdaParameters is exited.
func (s *BaseKotlinParserListener) ExitLambdaParameters(ctx *LambdaParametersContext) {}

// EnterLambdaParameter is called when production lambdaParameter is entered.
func (s *BaseKotlinParserListener) EnterLambdaParameter(ctx *LambdaParameterContext) {}

// ExitLambdaParameter is called when production lambdaParameter is exited.
func (s *BaseKotlinParserListener) ExitLambdaParameter(ctx *LambdaParameterContext) {}

// EnterObjectLiteral is called when production objectLiteral is entered.
func (s *BaseKotlinParserListener) EnterObjectLiteral(ctx *ObjectLiteralContext) {}

// ExitObjectLiteral is called when production objectLiteral is exited.
func (s *BaseKotlinParserListener) ExitObjectLiteral(ctx *ObjectLiteralContext) {}

// EnterCollectionLiteral is called when production collectionLiteral is entered.
func (s *BaseKotlinParserListener) EnterCollectionLiteral(ctx *CollectionLiteralContext) {}

// ExitCollectionLiteral is called when production collectionLiteral is exited.
func (s *BaseKotlinParserListener) ExitCollectionLiteral(ctx *CollectionLiteralContext) {}

// EnterThisExpression is called when production thisExpression is entered.
func (s *BaseKotlinParserListener) EnterThisExpression(ctx *ThisExpressionContext) {}

// ExitThisExpression is called when production thisExpression is exited.
func (s *BaseKotlinParserListener) ExitThisExpression(ctx *ThisExpressionContext) {}

// EnterSuperExpression is called when production superExpression is entered.
func (s *BaseKotlinParserListener) EnterSuperExpression(ctx *SuperExpressionContext) {}

// ExitSuperExpression is called when production superExpression is exited.
func (s *BaseKotlinParserListener) ExitSuperExpression(ctx *SuperExpressionContext) {}

// EnterConditionalExpression is called when production conditionalExpression is entered.
func (s *BaseKotlinParserListener) EnterConditionalExpression(ctx *ConditionalExpressionContext) {}

// ExitConditionalExpression is called when production conditionalExpression is exited.
func (s *BaseKotlinParserListener) ExitConditionalExpression(ctx *ConditionalExpressionContext) {}

// EnterIfExpression is called when production ifExpression is entered.
func (s *BaseKotlinParserListener) EnterIfExpression(ctx *IfExpressionContext) {}

// ExitIfExpression is called when production ifExpression is exited.
func (s *BaseKotlinParserListener) ExitIfExpression(ctx *IfExpressionContext) {}

// EnterControlStructureBody is called when production controlStructureBody is entered.
func (s *BaseKotlinParserListener) EnterControlStructureBody(ctx *ControlStructureBodyContext) {}

// ExitControlStructureBody is called when production controlStructureBody is exited.
func (s *BaseKotlinParserListener) ExitControlStructureBody(ctx *ControlStructureBodyContext) {}

// EnterWhenExpression is called when production whenExpression is entered.
func (s *BaseKotlinParserListener) EnterWhenExpression(ctx *WhenExpressionContext) {}

// ExitWhenExpression is called when production whenExpression is exited.
func (s *BaseKotlinParserListener) ExitWhenExpression(ctx *WhenExpressionContext) {}

// EnterWhenEntry is called when production whenEntry is entered.
func (s *BaseKotlinParserListener) EnterWhenEntry(ctx *WhenEntryContext) {}

// ExitWhenEntry is called when production whenEntry is exited.
func (s *BaseKotlinParserListener) ExitWhenEntry(ctx *WhenEntryContext) {}

// EnterWhenCondition is called when production whenCondition is entered.
func (s *BaseKotlinParserListener) EnterWhenCondition(ctx *WhenConditionContext) {}

// ExitWhenCondition is called when production whenCondition is exited.
func (s *BaseKotlinParserListener) ExitWhenCondition(ctx *WhenConditionContext) {}

// EnterRangeTest is called when production rangeTest is entered.
func (s *BaseKotlinParserListener) EnterRangeTest(ctx *RangeTestContext) {}

// ExitRangeTest is called when production rangeTest is exited.
func (s *BaseKotlinParserListener) ExitRangeTest(ctx *RangeTestContext) {}

// EnterTypeTest is called when production typeTest is entered.
func (s *BaseKotlinParserListener) EnterTypeTest(ctx *TypeTestContext) {}

// ExitTypeTest is called when production typeTest is exited.
func (s *BaseKotlinParserListener) ExitTypeTest(ctx *TypeTestContext) {}

// EnterTryExpression is called when production tryExpression is entered.
func (s *BaseKotlinParserListener) EnterTryExpression(ctx *TryExpressionContext) {}

// ExitTryExpression is called when production tryExpression is exited.
func (s *BaseKotlinParserListener) ExitTryExpression(ctx *TryExpressionContext) {}

// EnterCatchBlock is called when production catchBlock is entered.
func (s *BaseKotlinParserListener) EnterCatchBlock(ctx *CatchBlockContext) {}

// ExitCatchBlock is called when production catchBlock is exited.
func (s *BaseKotlinParserListener) ExitCatchBlock(ctx *CatchBlockContext) {}

// EnterFinallyBlock is called when production finallyBlock is entered.
func (s *BaseKotlinParserListener) EnterFinallyBlock(ctx *FinallyBlockContext) {}

// ExitFinallyBlock is called when production finallyBlock is exited.
func (s *BaseKotlinParserListener) ExitFinallyBlock(ctx *FinallyBlockContext) {}

// EnterLoopExpression is called when production loopExpression is entered.
func (s *BaseKotlinParserListener) EnterLoopExpression(ctx *LoopExpressionContext) {}

// ExitLoopExpression is called when production loopExpression is exited.
func (s *BaseKotlinParserListener) ExitLoopExpression(ctx *LoopExpressionContext) {}

// EnterForExpression is called when production forExpression is entered.
func (s *BaseKotlinParserListener) EnterForExpression(ctx *ForExpressionContext) {}

// ExitForExpression is called when production forExpression is exited.
func (s *BaseKotlinParserListener) ExitForExpression(ctx *ForExpressionContext) {}

// EnterWhileExpression is called when production whileExpression is entered.
func (s *BaseKotlinParserListener) EnterWhileExpression(ctx *WhileExpressionContext) {}

// ExitWhileExpression is called when production whileExpression is exited.
func (s *BaseKotlinParserListener) ExitWhileExpression(ctx *WhileExpressionContext) {}

// EnterDoWhileExpression is called when production doWhileExpression is entered.
func (s *BaseKotlinParserListener) EnterDoWhileExpression(ctx *DoWhileExpressionContext) {}

// ExitDoWhileExpression is called when production doWhileExpression is exited.
func (s *BaseKotlinParserListener) ExitDoWhileExpression(ctx *DoWhileExpressionContext) {}

// EnterJumpExpression is called when production jumpExpression is entered.
func (s *BaseKotlinParserListener) EnterJumpExpression(ctx *JumpExpressionContext) {}

// ExitJumpExpression is called when production jumpExpression is exited.
func (s *BaseKotlinParserListener) ExitJumpExpression(ctx *JumpExpressionContext) {}

// EnterCallableReference is called when production callableReference is entered.
func (s *BaseKotlinParserListener) EnterCallableReference(ctx *CallableReferenceContext) {}

// ExitCallableReference is called when production callableReference is exited.
func (s *BaseKotlinParserListener) ExitCallableReference(ctx *CallableReferenceContext) {}

// EnterAssignmentOperator is called when production assignmentOperator is entered.
func (s *BaseKotlinParserListener) EnterAssignmentOperator(ctx *AssignmentOperatorContext) {}

// ExitAssignmentOperator is called when production assignmentOperator is exited.
func (s *BaseKotlinParserListener) ExitAssignmentOperator(ctx *AssignmentOperatorContext) {}

// EnterEqualityOperation is called when production equalityOperation is entered.
func (s *BaseKotlinParserListener) EnterEqualityOperation(ctx *EqualityOperationContext) {}

// ExitEqualityOperation is called when production equalityOperation is exited.
func (s *BaseKotlinParserListener) ExitEqualityOperation(ctx *EqualityOperationContext) {}

// EnterComparisonOperator is called when production comparisonOperator is entered.
func (s *BaseKotlinParserListener) EnterComparisonOperator(ctx *ComparisonOperatorContext) {}

// ExitComparisonOperator is called when production comparisonOperator is exited.
func (s *BaseKotlinParserListener) ExitComparisonOperator(ctx *ComparisonOperatorContext) {}

// EnterInOperator is called when production inOperator is entered.
func (s *BaseKotlinParserListener) EnterInOperator(ctx *InOperatorContext) {}

// ExitInOperator is called when production inOperator is exited.
func (s *BaseKotlinParserListener) ExitInOperator(ctx *InOperatorContext) {}

// EnterIsOperator is called when production isOperator is entered.
func (s *BaseKotlinParserListener) EnterIsOperator(ctx *IsOperatorContext) {}

// ExitIsOperator is called when production isOperator is exited.
func (s *BaseKotlinParserListener) ExitIsOperator(ctx *IsOperatorContext) {}

// EnterAdditiveOperator is called when production additiveOperator is entered.
func (s *BaseKotlinParserListener) EnterAdditiveOperator(ctx *AdditiveOperatorContext) {}

// ExitAdditiveOperator is called when production additiveOperator is exited.
func (s *BaseKotlinParserListener) ExitAdditiveOperator(ctx *AdditiveOperatorContext) {}

// EnterMultiplicativeOperation is called when production multiplicativeOperation is entered.
func (s *BaseKotlinParserListener) EnterMultiplicativeOperation(ctx *MultiplicativeOperationContext) {
}

// ExitMultiplicativeOperation is called when production multiplicativeOperation is exited.
func (s *BaseKotlinParserListener) ExitMultiplicativeOperation(ctx *MultiplicativeOperationContext) {}

// EnterTypeOperation is called when production typeOperation is entered.
func (s *BaseKotlinParserListener) EnterTypeOperation(ctx *TypeOperationContext) {}

// ExitTypeOperation is called when production typeOperation is exited.
func (s *BaseKotlinParserListener) ExitTypeOperation(ctx *TypeOperationContext) {}

// EnterPrefixUnaryOperation is called when production prefixUnaryOperation is entered.
func (s *BaseKotlinParserListener) EnterPrefixUnaryOperation(ctx *PrefixUnaryOperationContext) {}

// ExitPrefixUnaryOperation is called when production prefixUnaryOperation is exited.
func (s *BaseKotlinParserListener) ExitPrefixUnaryOperation(ctx *PrefixUnaryOperationContext) {}

// EnterPostfixUnaryOperation is called when production postfixUnaryOperation is entered.
func (s *BaseKotlinParserListener) EnterPostfixUnaryOperation(ctx *PostfixUnaryOperationContext) {}

// ExitPostfixUnaryOperation is called when production postfixUnaryOperation is exited.
func (s *BaseKotlinParserListener) ExitPostfixUnaryOperation(ctx *PostfixUnaryOperationContext) {}

// EnterMemberAccessOperator is called when production memberAccessOperator is entered.
func (s *BaseKotlinParserListener) EnterMemberAccessOperator(ctx *MemberAccessOperatorContext) {}

// ExitMemberAccessOperator is called when production memberAccessOperator is exited.
func (s *BaseKotlinParserListener) ExitMemberAccessOperator(ctx *MemberAccessOperatorContext) {}

// EnterModifierList is called when production modifierList is entered.
func (s *BaseKotlinParserListener) EnterModifierList(ctx *ModifierListContext) {}

// ExitModifierList is called when production modifierList is exited.
func (s *BaseKotlinParserListener) ExitModifierList(ctx *ModifierListContext) {}

// EnterModifier is called when production modifier is entered.
func (s *BaseKotlinParserListener) EnterModifier(ctx *ModifierContext) {}

// ExitModifier is called when production modifier is exited.
func (s *BaseKotlinParserListener) ExitModifier(ctx *ModifierContext) {}

// EnterClassModifier is called when production classModifier is entered.
func (s *BaseKotlinParserListener) EnterClassModifier(ctx *ClassModifierContext) {}

// ExitClassModifier is called when production classModifier is exited.
func (s *BaseKotlinParserListener) ExitClassModifier(ctx *ClassModifierContext) {}

// EnterMemberModifier is called when production memberModifier is entered.
func (s *BaseKotlinParserListener) EnterMemberModifier(ctx *MemberModifierContext) {}

// ExitMemberModifier is called when production memberModifier is exited.
func (s *BaseKotlinParserListener) ExitMemberModifier(ctx *MemberModifierContext) {}

// EnterVisibilityModifier is called when production visibilityModifier is entered.
func (s *BaseKotlinParserListener) EnterVisibilityModifier(ctx *VisibilityModifierContext) {}

// ExitVisibilityModifier is called when production visibilityModifier is exited.
func (s *BaseKotlinParserListener) ExitVisibilityModifier(ctx *VisibilityModifierContext) {}

// EnterVarianceAnnotation is called when production varianceAnnotation is entered.
func (s *BaseKotlinParserListener) EnterVarianceAnnotation(ctx *VarianceAnnotationContext) {}

// ExitVarianceAnnotation is called when production varianceAnnotation is exited.
func (s *BaseKotlinParserListener) ExitVarianceAnnotation(ctx *VarianceAnnotationContext) {}

// EnterFunctionModifier is called when production functionModifier is entered.
func (s *BaseKotlinParserListener) EnterFunctionModifier(ctx *FunctionModifierContext) {}

// ExitFunctionModifier is called when production functionModifier is exited.
func (s *BaseKotlinParserListener) ExitFunctionModifier(ctx *FunctionModifierContext) {}

// EnterPropertyModifier is called when production propertyModifier is entered.
func (s *BaseKotlinParserListener) EnterPropertyModifier(ctx *PropertyModifierContext) {}

// ExitPropertyModifier is called when production propertyModifier is exited.
func (s *BaseKotlinParserListener) ExitPropertyModifier(ctx *PropertyModifierContext) {}

// EnterInheritanceModifier is called when production inheritanceModifier is entered.
func (s *BaseKotlinParserListener) EnterInheritanceModifier(ctx *InheritanceModifierContext) {}

// ExitInheritanceModifier is called when production inheritanceModifier is exited.
func (s *BaseKotlinParserListener) ExitInheritanceModifier(ctx *InheritanceModifierContext) {}

// EnterParameterModifier is called when production parameterModifier is entered.
func (s *BaseKotlinParserListener) EnterParameterModifier(ctx *ParameterModifierContext) {}

// ExitParameterModifier is called when production parameterModifier is exited.
func (s *BaseKotlinParserListener) ExitParameterModifier(ctx *ParameterModifierContext) {}

// EnterTypeParameterModifier is called when production typeParameterModifier is entered.
func (s *BaseKotlinParserListener) EnterTypeParameterModifier(ctx *TypeParameterModifierContext) {}

// ExitTypeParameterModifier is called when production typeParameterModifier is exited.
func (s *BaseKotlinParserListener) ExitTypeParameterModifier(ctx *TypeParameterModifierContext) {}

// EnterLabelDefinition is called when production labelDefinition is entered.
func (s *BaseKotlinParserListener) EnterLabelDefinition(ctx *LabelDefinitionContext) {}

// ExitLabelDefinition is called when production labelDefinition is exited.
func (s *BaseKotlinParserListener) ExitLabelDefinition(ctx *LabelDefinitionContext) {}

// EnterAnnotations is called when production annotations is entered.
func (s *BaseKotlinParserListener) EnterAnnotations(ctx *AnnotationsContext) {}

// ExitAnnotations is called when production annotations is exited.
func (s *BaseKotlinParserListener) ExitAnnotations(ctx *AnnotationsContext) {}

// EnterAnnotation is called when production annotation is entered.
func (s *BaseKotlinParserListener) EnterAnnotation(ctx *AnnotationContext) {}

// ExitAnnotation is called when production annotation is exited.
func (s *BaseKotlinParserListener) ExitAnnotation(ctx *AnnotationContext) {}

// EnterAnnotationList is called when production annotationList is entered.
func (s *BaseKotlinParserListener) EnterAnnotationList(ctx *AnnotationListContext) {}

// ExitAnnotationList is called when production annotationList is exited.
func (s *BaseKotlinParserListener) ExitAnnotationList(ctx *AnnotationListContext) {}

// EnterAnnotationUseSiteTarget is called when production annotationUseSiteTarget is entered.
func (s *BaseKotlinParserListener) EnterAnnotationUseSiteTarget(ctx *AnnotationUseSiteTargetContext) {
}

// ExitAnnotationUseSiteTarget is called when production annotationUseSiteTarget is exited.
func (s *BaseKotlinParserListener) ExitAnnotationUseSiteTarget(ctx *AnnotationUseSiteTargetContext) {}

// EnterUnescapedAnnotation is called when production unescapedAnnotation is entered.
func (s *BaseKotlinParserListener) EnterUnescapedAnnotation(ctx *UnescapedAnnotationContext) {}

// ExitUnescapedAnnotation is called when production unescapedAnnotation is exited.
func (s *BaseKotlinParserListener) ExitUnescapedAnnotation(ctx *UnescapedAnnotationContext) {}

// EnterIdentifier is called when production identifier is entered.
func (s *BaseKotlinParserListener) EnterIdentifier(ctx *IdentifierContext) {}

// ExitIdentifier is called when production identifier is exited.
func (s *BaseKotlinParserListener) ExitIdentifier(ctx *IdentifierContext) {}

// EnterSimpleIdentifier is called when production simpleIdentifier is entered.
func (s *BaseKotlinParserListener) EnterSimpleIdentifier(ctx *SimpleIdentifierContext) {}

// ExitSimpleIdentifier is called when production simpleIdentifier is exited.
func (s *BaseKotlinParserListener) ExitSimpleIdentifier(ctx *SimpleIdentifierContext) {}

// EnterSemi is called when production semi is entered.
func (s *BaseKotlinParserListener) EnterSemi(ctx *SemiContext) {}

// ExitSemi is called when production semi is exited.
func (s *BaseKotlinParserListener) ExitSemi(ctx *SemiContext) {}

// EnterAnysemi is called when production anysemi is entered.
func (s *BaseKotlinParserListener) EnterAnysemi(ctx *AnysemiContext) {}

// ExitAnysemi is called when production anysemi is exited.
func (s *BaseKotlinParserListener) ExitAnysemi(ctx *AnysemiContext) {}
