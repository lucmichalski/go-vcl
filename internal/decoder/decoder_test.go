package decoder

import (
	"reflect"
	"testing"

	"github.com/KeisukeYamashita/go-vcl/internal/ast"
	"github.com/KeisukeYamashita/go-vcl/internal/lexer"
	"github.com/KeisukeYamashita/go-vcl/internal/parser"
	"github.com/KeisukeYamashita/go-vcl/internal/schema"
)

func TestDecode(t *testing.T) {
	type TestStruct struct {
		Name string
	}

	testStruct := &TestStruct{}
	prog := &ast.Program{}

	testCases := []struct {
		input   interface{}
		program *ast.Program
	}{
		{testStruct, prog},
	}

	for n, tc := range testCases {
		if errs := Decode(tc.program, tc.input); len(errs) > 0 {
			t.Fatalf("decode failed with error[testcase:%d]", n)
		}
	}
}

func TestDecodeProgramToStruct_Attribute(t *testing.T) {
	type Root struct {
		X   int64  `vcl:"x"`
		API string `vcl:"api"`
	}

	testCases := []struct {
		input    string
		val      interface{}
		expected interface{}
	}{
		{`x = 1`, &Root{}, &Root{X: 1}},
		{`api = "localhost"`, &Root{}, &Root{API: "localhost"}},
	}

	for n, tc := range testCases {
		l := lexer.NewLexer(tc.input)
		p := parser.NewParser(l)
		program := p.ParseProgram()
		root := tc.val
		val := reflect.ValueOf(root).Elem()
		errs := decodeProgramToStruct(program, val)

		if len(errs) > 0 {
			t.Fatalf("decodeProgramToStruct has errors[testCase:%d], err:%v", n, errs)
		}

		if !reflect.DeepEqual(tc.val, tc.expected) {
			t.Fatalf("decodeProgramToStruct got wrong result[testCase:%d]", n)
		}
	}
}

func TestDecodeProgramToStruct_Block(t *testing.T) {
	type ACL struct {
		Type      string   `vcl:"type,label"`
		Name      string   `vcl:"name,label"`
		Endpoints []string `vcl:"endpoints,flat"`
	}

	type Sub struct {
		Type      string   `vcl:"type,label"`
		Endpoints []string `vcl:"endpoints,flat"` // Memo(KeisukeYamashita): Wont test inside of the block
	}

	type SubObj struct {
		Type string `vcl:"type,label"`
		Name string `vcl:"name,label"`
		Host string `vcl:".host"`
		IP   string `vcl:".ip"`
	}

	type RootSub struct {
		Subs []*SubObj `vcl:"sub,block"`
	}

	type Root struct {
		ACLs []*ACL `vcl:"acl,block"`
		Subs []*Sub `vcl:"sub,block"`
	}

	testCases := map[string]struct {
		input    string
		val      interface{}
		expected interface{}
	}{
		"with single block": {
			`acl local {
	"local";
	"localhost";
}`, &Root{}, &Root{Subs: []*Sub{}, ACLs: []*ACL{&ACL{Type: "local", Endpoints: []string{"local", "localhost"}}}},
		},
		"with two same block": {
			`acl local {
	"local";	
	"localhost";
}

acl remote {
	"remote";
}
`, &Root{}, &Root{Subs: []*Sub{}, ACLs: []*ACL{&ACL{Type: "local", Endpoints: []string{"local", "localhost"}}, &ACL{Type: "remote", Endpoints: []string{"remote"}}}},
		},
		"with two mixed block type": {
			`acl local {
	"local";	
	"localhost";
}

sub pipe_something {
	"inside_sub";
	"34.100.0.0"/23;
}
`, &Root{}, &Root{ACLs: []*ACL{&ACL{Type: "local", Endpoints: []string{"local", "localhost"}}}, Subs: []*Sub{&Sub{Type: "pipe_something", Endpoints: []string{"inside_sub", "\"34.100.0.0\"/23"}}}},
		},
		"with sub block": {
			`sub pipe_something {
	.host = "host";
	.ip = "ip";
}
`, &RootSub{}, &RootSub{Subs: []*SubObj{&SubObj{Type: "pipe_something", Host: "host", IP: "ip"}}},
		},
		"with multi label": {
			`sub pipe_something pipe_keke {
	.host = "host";
	.ip = "ip";
}
`, &RootSub{}, &RootSub{Subs: []*SubObj{&SubObj{Type: "pipe_something", Name: "pipe_keke", Host: "host", IP: "ip"}}},
		},
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			l := lexer.NewLexer(tc.input)
			p := parser.NewParser(l)
			program := p.ParseProgram()
			root := tc.val
			val := reflect.ValueOf(root).Elem()
			errs := decodeProgramToStruct(program, val)

			if len(errs) > 0 {
				t.Fatalf("decodeProgramToStruct_Block has errorr, err:%v", errs)
			}

			if !reflect.DeepEqual(tc.val, tc.expected) {
				t.Fatalf("decodeProgramToStruct_Block got wrong result, got:%#v", tc.val)
			}
		})
	}
}

func TestDecodeProgramToStruct_NestedBlock(t *testing.T) {
	type Probe struct {
		X int64 `vcl:"x"`
	}

	type Backend struct {
		Type  string `vcl:"type,label"`
		IP    string `vcl:".ip"`
		Probe *Probe `vcl:".probe,block"`
	}

	type Root struct {
		Backends []*Backend `vcl:"backend,block"`
	}

	testCases := map[string]struct {
		input    string
		val      interface{}
		expected interface{}
	}{
		"with nested simple block": {
			`backend remote {
	.ip = "localhost";
	.probe = {
		x = 10;
	};
}`, &Root{}, &Root{Backends: []*Backend{&Backend{Type: "remote", IP: "localhost", Probe: &Probe{X: 10}}}},
		},
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			l := lexer.NewLexer(tc.input)
			p := parser.NewParser(l)
			program := p.ParseProgram()
			root := tc.val
			val := reflect.ValueOf(root).Elem()
			errs := decodeProgramToStruct(program, val)

			if len(errs) > 0 {
				t.Fatalf("decodeProgramToStruct_Block has errorr, err:%v", errs)
			}

			// pp.Println(tc.val)

			if !reflect.DeepEqual(tc.val, tc.expected) {
				t.Fatalf("decodeProgramToStruct_Block got wrong result, got:%#v", tc.val)
			}
		})
	}
}

func TestImpliedBodySchema(t *testing.T) {
	type testBlock struct {
		Type       string `vcl:"type,label"`
		MiddelName string `vcl:"middelname"`
	}

	type testStruct struct {
		Type     string     `vcl:"type,label"`
		Name     string     `vcl:"name"`
		Resource *testBlock `vcl:"resource,block"`
	}

	input := &testStruct{
		Type: "my-type",
		Name: "keke",
		Resource: &testBlock{
			MiddelName: "middelName",
		},
	}

	testCases := []struct {
		input interface{}
	}{
		{input},
	}

	for n, tc := range testCases {
		file := impliedBodySchema(tc.input)
		bs := file.Body.(*schema.BodySchema)
		if len(bs.Attributes) != 1 {
			t.Fatalf("Attribute length wrong[testCase:%d], got:%d, want:%d", n, len(bs.Attributes), 1)
		}

		if len(bs.Blocks) != 1 {
			t.Fatalf("Block length wrong[testCase:%d], got:%d, want:%d", n, len(bs.Blocks), 1)
		}

		if len(bs.Blocks[0].LabelNames) != 1 {
			t.Fatalf("Block label are not expected[testCase:%d], got:%d, want:%d", n, len(bs.Blocks[0].LabelNames), 1)
		}
	}
}

func TestGetFieldTags(t *testing.T) {
	type testStruct struct {
		Type     string      `vcl:"type,label"`
		Name     string      `vcl:"name"` // implied attribute
		Resource interface{} `vcl:"resource,block"`
	}

	input := &testStruct{
		Type:     "my-type",
		Name:     "keke",
		Resource: "",
	}

	testCases := []struct {
		input *testStruct
	}{
		{input},
	}

	for n, tc := range testCases {
		ty := reflect.TypeOf(*tc.input)
		tags := getFieldTags(ty)

		if len(tags.Attributes) != 1 {
			t.Fatalf("Attribute length wrong[testCase:%d], got:%d, want:%d", n, len(tags.Attributes), 1)
		}

		if len(tags.Labels) != 1 {
			t.Fatalf("Labels length wrong[testCase:%d], got:%d, want:%d", n, len(tags.Labels), 1)
		}

		if len(tags.Blocks) != 1 {
			t.Fatalf("Blocks length wrong[testCase:%d], got:%d, want:%d", n, len(tags.Blocks), 1)
		}

	}
}
