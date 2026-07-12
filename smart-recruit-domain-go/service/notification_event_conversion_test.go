package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

func TestNotificationOutboxWritesUseDomainEventTypes(t *testing.T) {
	t.Parallel()

	files := []string{
		"application_service.go",
		"interview_service.go",
		"offer_service.go",
	}
	for _, fileName := range files {
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()
			fileSet := token.NewFileSet()
			file, err := parser.ParseFile(fileSet, fileName, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", fileName, err)
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "WriteEventTx" || len(call.Args) < 5 {
					return true
				}
				eventTypeLiteral, ok := call.Args[1].(*ast.BasicLit)
				if !ok || eventTypeLiteral.Kind != token.STRING {
					return true
				}
				eventType, err := strconv.Unquote(eventTypeLiteral.Value)
				if err != nil {
					t.Fatalf("unquote event type in %s: %v", fileName, err)
				}
				if eventType == "notification.create" || eventType == "email.send" {
					position := fileSet.Position(eventTypeLiteral.Pos())
					t.Fatalf("%s uses transport routing key as domain event type at %s", fileName, position)
				}
				return true
			})
		})
	}
}
