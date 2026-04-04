package lang

import (
	"strings"
	"testing"

	"github.com/j178/leetgo/leetcode"
)

// buildVoidQuestion returns a minimal QuestionData for a void in-place problem
// that mirrors problem 283 (moveZeroes): void moveZeroes(vector<int>& nums)
func buildVoidQuestion() *leetcode.QuestionData {
	return &leetcode.QuestionData{
		TitleSlug: "move-zeroes",
		MetaData: leetcode.MetaData{
			Name: "moveZeroes",
			Params: []leetcode.MetaDataParam{
				{Name: "nums", Type: "integer[]"},
			},
			Return: &leetcode.MetaDataReturn{Type: "void"},
			Output: &leetcode.MetaDataOutput{ParamIndex: 0},
		},
	}
}

// buildNonVoidQuestion returns a minimal QuestionData for a problem that returns a value
// that mirrors problem 1 (twoSum): vector<int> twoSum(vector<int>& nums, int target)
func buildNonVoidQuestion() *leetcode.QuestionData {
	return &leetcode.QuestionData{
		TitleSlug: "two-sum",
		MetaData: leetcode.MetaData{
			Name: "twoSum",
			Params: []leetcode.MetaDataParam{
				{Name: "nums", Type: "integer[]"},
				{Name: "target", Type: "integer"},
			},
			Return: &leetcode.MetaDataReturn{Type: "integer[]"},
		},
	}
}

func TestGenerateCallCode_VoidReturn(t *testing.T) {
	c := cpp{}
	q := buildVoidQuestion()
	code := c.generateCallCode(q)

	// Must NOT contain assignment to auto res
	if strings.Contains(code, "auto res") {
		t.Errorf("generateCallCode for void return should not assign to auto res, got:\n%s", code)
	}
	// Must call the method
	if !strings.Contains(code, "obj->moveZeroes") {
		t.Errorf("generateCallCode should call obj->moveZeroes, got:\n%s", code)
	}
}

func TestGenerateCallCode_NonVoidReturn(t *testing.T) {
	c := cpp{}
	q := buildNonVoidQuestion()
	code := c.generateCallCode(q)

	// Must contain assignment to auto res
	if !strings.Contains(code, "auto res") {
		t.Errorf("generateCallCode for non-void return should assign to auto res, got:\n%s", code)
	}
}

func TestGeneratePrintCode_VoidReturn_WithOutput(t *testing.T) {
	c := cpp{}
	q := buildVoidQuestion()
	code := c.generatePrintCode(q)

	// Must print the modified param (nums), not res
	if strings.Contains(code, "out_stream, res") {
		t.Errorf("generatePrintCode for void return should not print res, got:\n%s", code)
	}
	if !strings.Contains(code, "out_stream, nums") {
		t.Errorf("generatePrintCode for void return should print nums (Output.ParamIndex=0), got:\n%s", code)
	}
}

// buildVoidNoOutputQuestion returns a minimal QuestionData for a void method
// with no Output metadata, exercising the "null" fallback branch.
func buildVoidNoOutputQuestion() *leetcode.QuestionData {
	return &leetcode.QuestionData{
		TitleSlug: "void-no-output",
		MetaData: leetcode.MetaData{
			Name: "doSomething",
			Params: []leetcode.MetaDataParam{
				{Name: "x", Type: "integer"},
			},
			Return: &leetcode.MetaDataReturn{Type: "void"},
			// Output is nil — should trigger "null" fallback
		},
	}
}

func TestGeneratePrintCode_VoidReturn_NoOutput(t *testing.T) {
	c := cpp{}
	q := buildVoidNoOutputQuestion()
	code := c.generatePrintCode(q)

	// Must NOT print res or any param — should print "null"
	if strings.Contains(code, "out_stream, res") {
		t.Errorf("generatePrintCode for void without Output should not print res, got:\n%s", code)
	}
	if !strings.Contains(code, `"null"`) {
		t.Errorf("generatePrintCode for void without Output should print \"null\", got:\n%s", code)
	}
}

func TestGeneratePrintCode_NonVoidReturn(t *testing.T) {
	c := cpp{}
	q := buildNonVoidQuestion()
	code := c.generatePrintCode(q)

	// Must print res
	if !strings.Contains(code, "out_stream, res") {
		t.Errorf("generatePrintCode for non-void return should print res, got:\n%s", code)
	}
}
