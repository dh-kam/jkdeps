package parser

import (
	"testing"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

func TestBuildJavaSourceFileCapturesJavaSignatures(t *testing.T) {
	source := []byte(`package com.example;

import java.io.IOException;
import java.util.*;
import static java.util.Collections.emptyList;

public class Sample extends BaseType implements Handler, java.io.Closeable {
	private List<String> names;

	public Sample(List<String> names) throws IOException {}

	public Result process(Input input, String... tags) throws IOException, CustomError {
		return null;
	}

	interface Nested extends Runnable {}
}

record UserRecord(String name, Handler handler) implements java.io.Serializable {}
`)

	file, err := BuildJavaSourceFile(source, JavaGrammar20)
	if err != nil {
		t.Fatalf("BuildJavaSourceFile() error = %v", err)
	}

	if file.Package.Name != "com.example" {
		t.Fatalf("file.Package.Name = %q, want %q", file.Package.Name, "com.example")
	}
	if len(file.Imports) != 3 {
		t.Fatalf("len(file.Imports) = %d, want 3", len(file.Imports))
	}
	importByPath := make(map[string]ast.Import, len(file.Imports))
	for _, imp := range file.Imports {
		importByPath[imp.Path] = imp
	}
	if imp := importByPath["java.io.IOException"]; imp.IsStatic || imp.IsWildcard {
		t.Fatalf("java.io.IOException import = %+v", imp)
	}
	if imp := importByPath["java.util.Collections.emptyList"]; !imp.IsStatic {
		t.Fatalf("java.util.Collections.emptyList import = %+v", imp)
	}
	if imp := importByPath["java.util.*"]; !imp.IsWildcard {
		t.Fatalf("java.util.* import = %+v", imp)
	}

	if len(file.Declarations) != 2 {
		t.Fatalf("len(file.Declarations) = %d, want 2", len(file.Declarations))
	}

	sample, ok := file.Declarations[0].(*ast.ClassDeclaration)
	if !ok {
		t.Fatalf("file.Declarations[0] = %T, want *ast.ClassDeclaration", file.Declarations[0])
	}
	if sample.Name_ != "Sample" || sample.Kind_ != ast.KindClass {
		t.Fatalf("sample = %+v", sample)
	}
	if sample.SuperClass == nil || sample.SuperClass.Name != "BaseType" {
		t.Fatalf("sample.SuperClass = %+v, want BaseType", sample.SuperClass)
	}
	if len(sample.SuperInterfaces) != 2 {
		t.Fatalf("len(sample.SuperInterfaces) = %d, want 2", len(sample.SuperInterfaces))
	}
	if sample.SuperInterfaces[0].Name != "Handler" {
		t.Fatalf("sample.SuperInterfaces[0] = %+v, want Handler", sample.SuperInterfaces[0])
	}
	if sample.SuperInterfaces[1].Package != "java.io" || sample.SuperInterfaces[1].Name != "Closeable" {
		t.Fatalf("sample.SuperInterfaces[1] = %+v, want java.io.Closeable", sample.SuperInterfaces[1])
	}

	if len(sample.Members) != 4 {
		t.Fatalf("len(sample.Members) = %d, want 4", len(sample.Members))
	}

	field, ok := sample.Members[0].(*ast.VariableDeclaration)
	if !ok {
		t.Fatalf("sample.Members[0] = %T, want *ast.VariableDeclaration", sample.Members[0])
	}
	if field.Name != "names" || field.Type == nil || field.Type.Name != "List" {
		t.Fatalf("field = %+v", field)
	}
	if len(field.Type.TypeArguments) != 1 || field.Type.TypeArguments[0].Name != "String" {
		t.Fatalf("field.Type = %+v", field.Type)
	}

	constructor, ok := sample.Members[1].(*ast.Constructor)
	if !ok {
		t.Fatalf("sample.Members[1] = %T, want *ast.Constructor", sample.Members[1])
	}
	if len(constructor.Parameters) != 1 || constructor.Parameters[0].Type == nil || constructor.Parameters[0].Type.Name != "List" {
		t.Fatalf("constructor = %+v", constructor)
	}

	method, ok := sample.Members[2].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("sample.Members[2] = %T, want *ast.FunctionDeclaration", sample.Members[2])
	}
	if method.Name_ != "process" || method.ReturnType == nil || method.ReturnType.Name != "Result" {
		t.Fatalf("method = %+v", method)
	}
	if len(method.Parameters) != 2 {
		t.Fatalf("len(method.Parameters) = %d, want 2", len(method.Parameters))
	}
	if method.Parameters[0].Type == nil || method.Parameters[0].Type.Name != "Input" {
		t.Fatalf("method.Parameters[0] = %+v", method.Parameters[0])
	}
	if method.Parameters[1].Type == nil || method.Parameters[1].Type.Name != "String" || !method.Parameters[1].IsVarArg {
		t.Fatalf("method.Parameters[1] = %+v", method.Parameters[1])
	}
	if len(method.Throws) != 2 || method.Throws[0].Name != "IOException" || method.Throws[1].Name != "CustomError" {
		t.Fatalf("method.Throws = %+v", method.Throws)
	}

	nested, ok := sample.Members[3].(*ast.ClassDeclaration)
	if !ok {
		t.Fatalf("sample.Members[3] = %T, want *ast.ClassDeclaration", sample.Members[3])
	}
	if nested.Name_ != "Nested" || nested.Kind_ != ast.KindInterface {
		t.Fatalf("nested = %+v", nested)
	}
	if len(nested.SuperInterfaces) != 1 || nested.SuperInterfaces[0].Name != "Runnable" {
		t.Fatalf("nested.SuperInterfaces = %+v", nested.SuperInterfaces)
	}

	record, ok := file.Declarations[1].(*ast.ClassDeclaration)
	if !ok {
		t.Fatalf("file.Declarations[1] = %T, want *ast.ClassDeclaration", file.Declarations[1])
	}
	if record.Kind_ != ast.KindRecord || !record.IsRecord {
		t.Fatalf("record = %+v", record)
	}
	if len(record.RecordComponents) != 2 {
		t.Fatalf("len(record.RecordComponents) = %d, want 2", len(record.RecordComponents))
	}
	if record.RecordComponents[0].Name != "name" || record.RecordComponents[0].Type == nil || record.RecordComponents[0].Type.Name != "String" {
		t.Fatalf("record.RecordComponents[0] = %+v", record.RecordComponents[0])
	}
	if record.RecordComponents[1].Name != "handler" || record.RecordComponents[1].Type == nil || record.RecordComponents[1].Type.Name != "Handler" {
		t.Fatalf("record.RecordComponents[1] = %+v", record.RecordComponents[1])
	}
	if len(record.SuperInterfaces) != 1 || record.SuperInterfaces[0].Package != "java.io" || record.SuperInterfaces[0].Name != "Serializable" {
		t.Fatalf("record.SuperInterfaces = %+v", record.SuperInterfaces)
	}
}

func TestParseJavaTypeReferenceText(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		wantPackage  string
		wantName     string
		wantArgCount int
	}{
		{
			name:         "qualified generic type",
			text:         "java.util.List<com.example.Item>",
			wantPackage:  "java.util",
			wantName:     "List",
			wantArgCount: 1,
		},
		{
			name:         "nested type",
			text:         "Map.Entry<String,Integer>",
			wantPackage:  "",
			wantName:     "Map.Entry",
			wantArgCount: 2,
		},
		{
			name:         "wildcard bound",
			text:         "?extendscom.example.Value",
			wantPackage:  "com.example",
			wantName:     "Value",
			wantArgCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJavaTypeReferenceText(tt.text)
			if got == nil {
				t.Fatal("parseJavaTypeReferenceText() = nil")
			}
			if got.Package != tt.wantPackage || got.Name != tt.wantName {
				t.Fatalf("parseJavaTypeReferenceText(%q) = %+v, want package=%q name=%q", tt.text, got, tt.wantPackage, tt.wantName)
			}
			if len(got.TypeArguments) != tt.wantArgCount {
				t.Fatalf("len(got.TypeArguments) = %d, want %d", len(got.TypeArguments), tt.wantArgCount)
			}
		})
	}
}

func TestBuildJavaSourceFileCapturesJavaBodyReferences(t *testing.T) {
	source := []byte(`package com.example;

import java.util.Collections;

class Sample {
  void run() {
    new Created();
    new java.util.ArrayList<String>();
    java.util.function.Supplier<java.util.ArrayList<String>> creator = java.util.ArrayList::new;
    java.util.function.Supplier<java.util.List<String>> empty = java.util.Collections::emptyList;
    java.util.Map<String, Created> local = null;
    Collections.emptyList();
    java.util.Objects.requireNonNull(this);
    Class<?> clazz = java.lang.String.class;
    Object obj = local;
    java.util.List<String> casted = (java.util.List<String>) obj;
    try {
      Collections.emptyList();
    } catch (java.io.IOException | CustomError ex) {
    }
    if (obj instanceof java.util.Map<String, ?> map) {
    }
  }
}
`)

	file, err := BuildJavaSourceFile(source, JavaGrammar20)
	if err != nil {
		t.Fatalf("BuildJavaSourceFile() error = %v", err)
	}

	got := map[ast.ReferenceUsageKind]map[string]struct{}{}
	for _, ref := range file.References {
		if got[ref.Kind] == nil {
			got[ref.Kind] = map[string]struct{}{}
		}
		got[ref.Kind][ref.Path] = struct{}{}
	}

	if _, ok := got[ast.ReferenceUsageConstructorCall]["Created"]; !ok {
		t.Fatalf("missing constructor_call Created in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageConstructorCall]["java.util.ArrayList"]; !ok {
		t.Fatalf("missing constructor_call java.util.ArrayList in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageLocalVariableType]["java.util.Map"]; !ok {
		t.Fatalf("missing local_variable_type java.util.Map in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageQualifiedMethodCall]["Collections"]; !ok {
		t.Fatalf("missing qualified_method_call Collections in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageQualifiedMethodCall]["java.util.Objects"]; !ok {
		t.Fatalf("missing qualified_method_call java.util.Objects in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageClassLiteral]["java.lang.String"]; !ok {
		t.Fatalf("missing class_literal java.lang.String in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageMethodReference]["java.util.Collections"]; !ok {
		t.Fatalf("missing method_reference java.util.Collections in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageConstructorReference]["java.util.ArrayList"]; !ok {
		t.Fatalf("missing constructor_reference java.util.ArrayList in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageCastType]["java.util.List"]; !ok {
		t.Fatalf("missing cast_type java.util.List in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageInstanceofType]["java.util.Map"]; !ok {
		t.Fatalf("missing instanceof_type java.util.Map in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageCatchType]["java.io.IOException"]; !ok {
		t.Fatalf("missing catch_type java.io.IOException in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageCatchType]["CustomError"]; !ok {
		t.Fatalf("missing catch_type CustomError in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageTypeArgument]["String"]; !ok {
		t.Fatalf("missing type_argument String in references: %+v", file.References)
	}
	if _, ok := got[ast.ReferenceUsageTypeArgument]["Created"]; !ok {
		t.Fatalf("missing type_argument Created in references: %+v", file.References)
	}
}
