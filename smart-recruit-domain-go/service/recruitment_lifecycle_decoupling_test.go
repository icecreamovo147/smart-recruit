package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestInterviewAndOfferUseRecruitmentLifecycleProcessManager(t *testing.T) {
	t.Parallel()

	forbiddenWrites := map[string]bool{
		"UpdateStatusAnyWithTx":   true,
		"CreateTransition":        true,
		"CloseCurrentRoundWithTx": true,
	}
	files := []string{"interview_service.go", "offer_service.go"}
	for _, fileName := range files {
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()

			fileSet := token.NewFileSet()
			file, err := parser.ParseFile(fileSet, fileName, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", fileName, err)
			}

			managerCalls := 0
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if selector.Sel.Name == "ApplyTransitionTx" {
					managerCalls++
				}
				if forbiddenWrites[selector.Sel.Name] {
					position := fileSet.Position(selector.Sel.Pos())
					t.Fatalf("%s performs direct recruitment lifecycle write %s at %s", fileName, selector.Sel.Name, position)
				}
				return true
			})
			if managerCalls == 0 {
				t.Fatalf("%s does not use RecruitmentLifecycleProcessManager", fileName)
			}
		})
	}
}
