package jmap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// call is one outgoing method invocation under construction.
type call struct {
	method string
	args   any
}

// batch builds a single JMAP request from one or more method calls and
// returns the parsed response. Call IDs are "c0", "c1", ... in order.
func (c *Client) batch(ctx context.Context, op string, using []string, calls ...call) (*wire.Response, error) {
	invocations := make([]wire.Invocation, len(calls))
	for i, cl := range calls {
		args, err := json.Marshal(cl.args)
		if err != nil {
			return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
		}
		invocations[i] = wire.Invocation{
			Name:   cl.method,
			Args:   args,
			CallID: "c" + strconv.Itoa(i),
		}
	}

	reqBody, err := json.Marshal(wire.Request{Using: using, MethodCalls: invocations})
	if err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}

	apiURL, _, _, _ := c.sessionURLs()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if err := c.auth.Authenticate(httpReq); err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeAuth, Err: err}
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResp))
	if err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: err}
	}

	if resp.StatusCode != http.StatusOK {
		code := httpStatusCode(resp.StatusCode)
		if code == mail.CodeAuth {
			auditEvent(ctx, slog.LevelWarn, "auth_failed",
				slog.String("op", op),
				slog.Int("status", resp.StatusCode))
		}
		// Request-level errors are RFC 7807 problem documents.
		var problem wire.Problem
		if json.Unmarshal(body, &problem) == nil && problem.Type != "" {
			return nil, &mail.Error{Op: op, Code: code, Err: &problem}
		}
		return nil, &mail.Error{Op: op, Code: code, Err: fmt.Errorf("status %d", resp.StatusCode)}
	}

	var parsed wire.Response
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}
	return &parsed, nil
}

// result locates the response for callID, decodes it into out, and maps
// method-level "error" invocations onto typed mail errors. expect is the
// method name the response must carry (e.g. "Email/get").
func (c *Client) result(op string, resp *wire.Response, callID, expect string, out any) error {
	inv := resp.Find(callID)
	if inv == nil {
		return &mail.Error{Op: op, Code: mail.CodeProtocol, Err: fmt.Errorf("no response for call %q", callID)}
	}
	if inv.Name == "error" {
		var methodErr wire.MethodError
		if err := json.Unmarshal(inv.Args, &methodErr); err != nil {
			return &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
		}
		return &mail.Error{Op: op, Code: methodErrorCode(methodErr.Type), Err: &methodErr}
	}
	if inv.Name != expect {
		return &mail.Error{Op: op, Code: mail.CodeProtocol, Err: fmt.Errorf("unexpected method %q, want %q", inv.Name, expect)}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(inv.Args, out); err != nil {
		return &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}
	return nil
}

// methodErrorCode maps RFC 8620 method-level error types onto
// mail.ErrorCode classes.
func methodErrorCode(typ string) mail.ErrorCode {
	switch typ {
	case "forbidden", "accountNotFound", "accountReadOnly":
		return mail.CodeAuth
	case "notFound":
		return mail.CodeNotFound
	case "overQuota", "tooLarge", "requestTooLarge":
		return mail.CodeTooLarge
	case "rateLimit", "limit", "serverUnavailable":
		return mail.CodeUnavailable
	case "unknownCapability", "unknownMethod", "cannotCalculateChanges", "unsupportedSort", "unsupportedFilter":
		return mail.CodeUnsupported
	default:
		return mail.CodeProtocol
	}
}

// setErrorCode maps a per-object SetError onto a mail.ErrorCode.
func setErrorCode(typ string) mail.ErrorCode {
	switch typ {
	case "notFound":
		return mail.CodeNotFound
	case "overQuota", "tooLarge":
		return mail.CodeTooLarge
	case "forbidden", "forbiddenFrom", "forbiddenMailFrom", "forbiddenToSend":
		return mail.CodeAuth
	default:
		return mail.CodeProtocol
	}
}

// firstSetError returns one representative error from a Foo/set failure
// map, or nil when the map is empty.
func firstSetError(op string, failed map[string]wire.SetError) error {
	for id, se := range failed {
		desc := se.Type
		if se.Description != "" {
			desc += ": " + se.Description
		}
		return &mail.Error{
			Op:   op,
			Code: setErrorCode(se.Type),
			Err:  fmt.Errorf("object %q: %s", id, desc),
		}
	}
	return nil
}
