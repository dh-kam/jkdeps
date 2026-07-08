package mixedgraph

import astpkg "github.com/dh-kam/jkdeps/internal/ast"

func extractJavaReferences(sourceFile *astpkg.SourceFile) []Reference {
	if sourceFile == nil {
		return nil
	}

	var refs []Reference
	for _, decl := range sourceFile.Declarations {
		classDecl, ok := decl.(*astpkg.ClassDeclaration)
		if !ok {
			continue
		}
		refs = appendClassReferences(refs, classDecl)
	}
	for _, ref := range sourceFile.References {
		mappedKind := mapASTReferenceKind(ref.Kind)
		if mappedKind == "" {
			continue
		}
		if path := normalizePath(ref.Path); path != "" {
			refs = append(refs, Reference{
				Path: path,
				Kind: mappedKind,
			})
		}
	}
	return refs
}

func mapASTReferenceKind(kind astpkg.ReferenceUsageKind) ReferenceKind {
	switch kind {
	case astpkg.ReferenceUsageConstructorCall:
		return ReferenceKindConstructorCall
	case astpkg.ReferenceUsageQualifiedMethodCall:
		return ReferenceKindQualifiedMethodCall
	case astpkg.ReferenceUsageClassLiteral:
		return ReferenceKindClassLiteral
	case astpkg.ReferenceUsageLocalVariableType:
		return ReferenceKindLocalVariableType
	case astpkg.ReferenceUsageCatchType:
		return ReferenceKindCatchType
	case astpkg.ReferenceUsageTypeArgument:
		return ReferenceKindTypeArgument
	case astpkg.ReferenceUsageMethodReference:
		return ReferenceKindMethodReference
	case astpkg.ReferenceUsageConstructorReference:
		return ReferenceKindConstructorReference
	case astpkg.ReferenceUsageCastType:
		return ReferenceKindCastType
	case astpkg.ReferenceUsageInstanceofType:
		return ReferenceKindInstanceofType
	default:
		return ""
	}
}

func appendClassReferences(refs []Reference, decl *astpkg.ClassDeclaration) []Reference {
	if decl == nil {
		return refs
	}

	refs = appendTypeReference(refs, ReferenceKindExtends, decl.SuperClass)
	for i := range decl.SuperInterfaces {
		refs = appendTypeReference(refs, ReferenceKindImplements, &decl.SuperInterfaces[i])
	}
	for i := range decl.RecordComponents {
		refs = appendTypeReference(refs, ReferenceKindFieldType, decl.RecordComponents[i].Type)
	}

	for _, member := range decl.Members {
		switch typed := member.(type) {
		case *astpkg.VariableDeclaration:
			refs = appendTypeReference(refs, ReferenceKindFieldType, typed.Type)
		case *astpkg.FunctionDeclaration:
			refs = appendTypeReference(refs, ReferenceKindMethodReturn, typed.ReturnType)
			for i := range typed.Parameters {
				refs = appendTypeReference(refs, ReferenceKindMethodParameter, typed.Parameters[i].Type)
			}
			for i := range typed.Throws {
				refs = appendTypeReference(refs, ReferenceKindThrows, &typed.Throws[i])
			}
		case *astpkg.Constructor:
			for i := range typed.Parameters {
				refs = appendTypeReference(refs, ReferenceKindConstructorParameter, typed.Parameters[i].Type)
			}
		case *astpkg.ClassDeclaration:
			refs = appendClassReferences(refs, typed)
		}
	}
	return refs
}

func appendTypeReference(refs []Reference, kind ReferenceKind, ref *astpkg.TypeReference) []Reference {
	if ref == nil || shouldSkipTypeReference(ref) {
		return refs
	}

	if path := astTypeReferencePath(ref); path != "" {
		refs = append(refs, Reference{Path: path, Kind: kind})
	}
	for i := range ref.TypeArguments {
		refs = appendTypeReference(refs, ReferenceKindTypeArgument, &ref.TypeArguments[i])
	}
	return refs
}

func astTypeReferencePath(ref *astpkg.TypeReference) string {
	if ref == nil || ref.Name == "" {
		return ""
	}
	if ref.Package == "" {
		return ref.Name
	}
	return ref.Package + "." + ref.Name
}

func shouldSkipTypeReference(ref *astpkg.TypeReference) bool {
	if ref == nil || ref.Name == "" {
		return true
	}
	switch ref.Name {
	case "boolean", "byte", "char", "double", "float", "int", "long", "short", "void":
		return true
	}
	return len(ref.Name) == 1 && ref.Package == "" && ref.Name[0] >= 'A' && ref.Name[0] <= 'Z'
}
