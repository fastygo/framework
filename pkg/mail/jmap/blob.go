package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// Attachment implements mail.Client. partID is the blob ID reported in
// AttachmentInfo.PartID; content streams directly from the download
// endpoint and is never buffered.
func (c *Client) Attachment(ctx context.Context, messageID, partID string) (io.ReadCloser, mail.AttachmentInfo, error) {
	const op = "jmap: blob download"
	var info mail.AttachmentInfo

	// Validate the part against the message so a caller cannot fetch
	// arbitrary blobs with a guessed ID/filename combination.
	msg, err := c.Message(ctx, messageID)
	if err != nil {
		return nil, info, err
	}
	found := false
	for _, att := range msg.Attachments {
		if att.PartID == partID {
			info = att
			found = true
			break
		}
	}
	if !found {
		return nil, info, &mail.Error{Op: op, Code: mail.CodeNotFound, Err: fmt.Errorf("part %q not in message %q", partID, messageID)}
	}

	_, downloadTmpl, _, _ := c.sessionURLs()
	if downloadTmpl == "" {
		return nil, info, &mail.Error{Op: op, Code: mail.CodeUnsupported, Err: fmt.Errorf("session has no downloadUrl")}
	}

	name := info.Filename
	if name == "" {
		name = "attachment"
	}
	contentType := info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	downloadURL := expandURITemplate(downloadTmpl, map[string]string{
		"accountId": c.account(),
		"blobId":    partID,
		"name":      name,
		"type":      contentType,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, info, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}
	if err := c.auth.Authenticate(req); err != nil {
		return nil, info, &mail.Error{Op: op, Code: mail.CodeAuth, Err: err}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, info, &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: err}
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, info, &mail.Error{Op: op, Code: httpStatusCode(resp.StatusCode), Err: fmt.Errorf("status %d", resp.StatusCode)}
	}
	return resp.Body, info, nil
}

// uploadBlob streams r to the upload endpoint and returns the blob ID.
func (c *Client) uploadBlob(ctx context.Context, contentType string, r io.Reader) (*wire.UploadResponse, error) {
	const op = "jmap: blob upload"

	_, _, uploadTmpl, _ := c.sessionURLs()
	if uploadTmpl == "" {
		return nil, &mail.Error{Op: op, Code: mail.CodeUnsupported, Err: fmt.Errorf("session has no uploadUrl")}
	}
	uploadURL := expandURITemplate(uploadTmpl, map[string]string{"accountId": c.account()})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, r)
	if err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}
	req.Header.Set("Content-Type", contentType)
	if err := c.auth.Authenticate(req); err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeAuth, Err: err}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResp))
	if err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: err}
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, &mail.Error{Op: op, Code: httpStatusCode(resp.StatusCode), Err: fmt.Errorf("status %d", resp.StatusCode)}
	}

	var uploaded wire.UploadResponse
	if err := json.Unmarshal(body, &uploaded); err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}
	return &uploaded, nil
}

// expandURITemplate substitutes {name} placeholders per the level-1 URI
// templates JMAP session URLs use (RFC 8620 §2).
func expandURITemplate(tmpl string, vars map[string]string) string {
	out := tmpl
	for key, val := range vars {
		out = strings.ReplaceAll(out, "{"+key+"}", url.PathEscape(val))
	}
	return out
}
