// Package wire holds the raw JMAP (RFC 8620 / RFC 8621) JSON shapes used
// by the jmap transport. Nothing here is part of the framework's public
// API; the parent package maps these types onto pkg/mail domain types.
package wire

import (
	"encoding/json"
	"fmt"
)

// Capability URNs the transport understands.
const (
	CapCore       = "urn:ietf:params:jmap:core"
	CapMail       = "urn:ietf:params:jmap:mail"
	CapSubmission = "urn:ietf:params:jmap:submission"
)

// Invocation is one method call or response: the three-element JSON array
// [name, arguments, callId] defined by RFC 8620 §3.2.
type Invocation struct {
	// Name is the method name, e.g. "Email/query", or "error" for a
	// method-level failure response.
	Name string
	// Args is the unparsed arguments object.
	Args json.RawMessage
	// CallID correlates a response with its request invocation.
	CallID string
}

// MarshalJSON encodes the invocation as [name, args, callId].
func (i Invocation) MarshalJSON() ([]byte, error) {
	args := i.Args
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	return json.Marshal([3]json.RawMessage{
		mustJSON(i.Name),
		args,
		mustJSON(i.CallID),
	})
}

// UnmarshalJSON decodes a [name, args, callId] triple.
func (i *Invocation) UnmarshalJSON(data []byte) error {
	var parts []json.RawMessage
	if err := json.Unmarshal(data, &parts); err != nil {
		return err
	}
	if len(parts) != 3 {
		return fmt.Errorf("wire: invocation has %d elements, want 3", len(parts))
	}
	if err := json.Unmarshal(parts[0], &i.Name); err != nil {
		return fmt.Errorf("wire: invocation name: %w", err)
	}
	i.Args = parts[1]
	if err := json.Unmarshal(parts[2], &i.CallID); err != nil {
		return fmt.Errorf("wire: invocation callId: %w", err)
	}
	return nil
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		// Strings always marshal; this cannot happen.
		panic(err)
	}
	return b
}

// Request is the top-level JMAP API request envelope (RFC 8620 §3.3).
type Request struct {
	Using       []string     `json:"using"`
	MethodCalls []Invocation `json:"methodCalls"`
}

// Response is the top-level JMAP API response envelope (RFC 8620 §3.4).
type Response struct {
	MethodResponses []Invocation `json:"methodResponses"`
	SessionState    string       `json:"sessionState"`
}

// Find returns the first method response with the given callID, or nil.
func (r *Response) Find(callID string) *Invocation {
	for idx := range r.MethodResponses {
		if r.MethodResponses[idx].CallID == callID {
			return &r.MethodResponses[idx]
		}
	}
	return nil
}

// MethodError is a method-level error response argument object
// (RFC 8620 §3.6.2).
type MethodError struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Error implements the error interface.
func (e *MethodError) Error() string {
	if e.Description == "" {
		return "jmap method error: " + e.Type
	}
	return "jmap method error: " + e.Type + ": " + e.Description
}

// Problem is an RFC 7807 problem-details object returned for
// request-level errors (RFC 8620 §3.6.1).
type Problem struct {
	Type   string `json:"type"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// Error implements the error interface.
func (p *Problem) Error() string {
	return fmt.Sprintf("jmap request error: %s (status %d): %s", p.Type, p.Status, p.Detail)
}

// Session is the JMAP session resource (RFC 8620 §2).
type Session struct {
	Capabilities    map[string]json.RawMessage `json:"capabilities"`
	Accounts        map[string]Account         `json:"accounts"`
	PrimaryAccounts map[string]string          `json:"primaryAccounts"`
	APIURL          string                     `json:"apiUrl"`
	DownloadURL     string                     `json:"downloadUrl"`
	UploadURL       string                     `json:"uploadUrl"`
	EventSourceURL  string                     `json:"eventSourceUrl"`
	State           string                     `json:"state"`
}

// Account is one account entry in the session resource.
type Account struct {
	Name                string                     `json:"name"`
	IsPersonal          bool                       `json:"isPersonal"`
	IsReadOnly          bool                       `json:"isReadOnly"`
	AccountCapabilities map[string]json.RawMessage `json:"accountCapabilities"`
}

// CoreCapability is the urn:ietf:params:jmap:core capability object.
type CoreCapability struct {
	MaxSizeUpload         int64 `json:"maxSizeUpload"`
	MaxSizeRequest        int64 `json:"maxSizeRequest"`
	MaxCallsInRequest     int64 `json:"maxCallsInRequest"`
	MaxObjectsInGet       int64 `json:"maxObjectsInGet"`
	MaxObjectsInSet       int64 `json:"maxObjectsInSet"`
	MaxConcurrentRequests int64 `json:"maxConcurrentRequests"`
}

// SubmissionCapability is the urn:ietf:params:jmap:submission account
// capability object.
type SubmissionCapability struct {
	MaxDelayedSend int64 `json:"maxDelayedSend"`
}
