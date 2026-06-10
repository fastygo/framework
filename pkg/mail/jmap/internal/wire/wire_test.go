package wire

import (
	"encoding/json"
	"testing"
)

func TestInvocationRoundTrip(t *testing.T) {
	in := Invocation{
		Name:   "Email/query",
		Args:   json.RawMessage(`{"accountId":"a1"}`),
		CallID: "c0",
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `["Email/query",{"accountId":"a1"},"c0"]`
	if string(data) != want {
		t.Errorf("marshal: got %s, want %s", data, want)
	}

	var out Invocation
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Name != in.Name || out.CallID != in.CallID || string(out.Args) != string(in.Args) {
		t.Errorf("round trip mismatch: %+v", out)
	}
}

func TestInvocationEmptyArgs(t *testing.T) {
	data, err := json.Marshal(Invocation{Name: "Core/echo", CallID: "c1"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `["Core/echo",{},"c1"]` {
		t.Errorf("empty args: got %s", data)
	}
}

func TestInvocationUnmarshalErrors(t *testing.T) {
	cases := []string{
		`["only-two","args"]`,
		`"not-an-array"`,
		`[123,{},"c0"]`,
		`["m",{},123]`,
	}
	for _, c := range cases {
		var inv Invocation
		if err := json.Unmarshal([]byte(c), &inv); err == nil {
			t.Errorf("expected error for %s", c)
		}
	}
}

func TestResponseFind(t *testing.T) {
	resp := Response{MethodResponses: []Invocation{
		{Name: "Email/query", CallID: "c0"},
		{Name: "Email/get", CallID: "c1"},
	}}
	if got := resp.Find("c1"); got == nil || got.Name != "Email/get" {
		t.Errorf("Find(c1): got %+v", got)
	}
	if got := resp.Find("missing"); got != nil {
		t.Errorf("Find(missing): got %+v", got)
	}
}

func TestErrorStrings(t *testing.T) {
	me := &MethodError{Type: "invalidArguments", Description: "bad filter"}
	if me.Error() != "jmap method error: invalidArguments: bad filter" {
		t.Errorf("MethodError: got %q", me.Error())
	}
	bare := &MethodError{Type: "serverFail"}
	if bare.Error() != "jmap method error: serverFail" {
		t.Errorf("MethodError bare: got %q", bare.Error())
	}
	p := &Problem{Type: "urn:ietf:params:jmap:error:notRequest", Status: 400, Detail: "nope"}
	if p.Error() == "" {
		t.Errorf("Problem: empty error string")
	}
}
