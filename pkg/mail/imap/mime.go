package imap

import (
	"bytes"
	"fmt"
	"io"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/emersion/go-message"
	gomail "github.com/emersion/go-message/mail"
	"github.com/fastygo/framework/pkg/mail"
)

func buildMIME(draft mail.Draft) ([]byte, string, error) {
	if err := validateDraft(draft); err != nil {
		return nil, "", err
	}

	var header gomail.Header
	header.Set("MIME-Version", "1.0")
	header.SetDate(time.Now())
	header.SetAddressList("From", messageAddresses([]mail.Address{draft.From}))
	header.SetAddressList("To", messageAddresses(draft.To))
	header.SetAddressList("Cc", messageAddresses(draft.Cc))
	header.SetSubject(draft.Subject)
	header.SetMsgIDList("In-Reply-To", trimMessageIDs(draft.InReplyTo))
	header.SetMsgIDList("References", trimMessageIDs(draft.References))
	hostname := messageIDHostname(draft.From.Email)
	if err := header.GenerateMessageIDWithHostname(hostname); err != nil {
		return nil, "", err
	}
	messageID, err := header.MessageID()
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	switch {
	case len(draft.Attachments) == 0 && draft.HTMLBody == "":
		header.SetContentType("text/plain", map[string]string{"charset": "utf-8"})
		writer, err := gomail.CreateSingleInlineWriter(&buf, header)
		if err != nil {
			return nil, "", err
		}
		if _, err := io.WriteString(writer, draft.TextBody); err != nil {
			_ = writer.Close()
			return nil, "", err
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
	case len(draft.Attachments) == 0:
		writer, err := gomail.CreateInlineWriter(&buf, header)
		if err != nil {
			return nil, "", err
		}
		if err := writeAlternative(writer, draft.TextBody, draft.HTMLBody); err != nil {
			_ = writer.Close()
			return nil, "", err
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
	default:
		writer, err := gomail.CreateWriter(&buf, header)
		if err != nil {
			return nil, "", err
		}
		if err := writeMixedBodies(writer, draft.TextBody, draft.HTMLBody); err != nil {
			_ = writer.Close()
			return nil, "", err
		}
		for _, attachment := range draft.Attachments {
			if err := writeOutgoingAttachment(writer, attachment); err != nil {
				_ = writer.Close()
				return nil, "", err
			}
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
	}
	return buf.Bytes(), trimMessageID(messageID), nil
}

func validateDraft(draft mail.Draft) error {
	if err := validateAddress(draft.From); err != nil {
		return fmt.Errorf("invalid From address: %w", err)
	}
	recipients := append(append(append([]mail.Address{}, draft.To...), draft.Cc...), draft.Bcc...)
	if len(recipients) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	for _, address := range recipients {
		if err := validateAddress(address); err != nil {
			return fmt.Errorf("invalid recipient: %w", err)
		}
	}
	if draft.TextBody == "" && draft.HTMLBody == "" {
		return fmt.Errorf("text or HTML body is required")
	}
	for _, attachment := range draft.Attachments {
		if attachment.Filename == "" {
			return fmt.Errorf("attachment filename is required")
		}
		if attachment.ContentType == "" {
			return fmt.Errorf("attachment content type is required")
		}
		if attachment.Open == nil {
			return fmt.Errorf("attachment Open is required")
		}
	}
	return nil
}

func validateAddress(address mail.Address) error {
	if address.Email == "" || strings.ContainsAny(address.Email, "\r\n") || strings.ContainsAny(address.Name, "\r\n") {
		return fmt.Errorf("malformed address")
	}
	parsed, err := netmail.ParseAddress(address.String())
	if err != nil {
		return err
	}
	if parsed.Address != address.Email {
		return fmt.Errorf("address does not round-trip")
	}
	return nil
}

func messageAddresses(input []mail.Address) []*gomail.Address {
	out := make([]*gomail.Address, len(input))
	for i := range input {
		out[i] = &gomail.Address{Name: input[i].Name, Address: input[i].Email}
	}
	return out
}

func messageIDHostname(address string) string {
	if at := strings.LastIndexByte(address, '@'); at >= 0 && at < len(address)-1 {
		return address[at+1:]
	}
	return "localhost"
}

func inlineHeader(contentType string) gomail.InlineHeader {
	var header message.Header
	header.SetContentType(contentType, map[string]string{"charset": "utf-8"})
	return gomail.InlineHeader{Header: header}
}

func writeAlternative(writer *gomail.InlineWriter, text, html string) error {
	if text != "" {
		part, err := writer.CreatePart(inlineHeader("text/plain"))
		if err != nil {
			return err
		}
		if _, err := io.WriteString(part, text); err != nil {
			_ = part.Close()
			return err
		}
		if err := part.Close(); err != nil {
			return err
		}
	}
	if html != "" {
		part, err := writer.CreatePart(inlineHeader("text/html"))
		if err != nil {
			return err
		}
		if _, err := io.WriteString(part, html); err != nil {
			_ = part.Close()
			return err
		}
		if err := part.Close(); err != nil {
			return err
		}
	}
	return nil
}

func writeMixedBodies(writer *gomail.Writer, text, html string) error {
	if text != "" && html != "" {
		inline, err := writer.CreateInline()
		if err != nil {
			return err
		}
		if err := writeAlternative(inline, text, html); err != nil {
			_ = inline.Close()
			return err
		}
		return inline.Close()
	}
	contentType, body := "text/plain", text
	if html != "" {
		contentType, body = "text/html", html
	}
	part, err := writer.CreateSingleInline(inlineHeader(contentType))
	if err != nil {
		return err
	}
	if _, err := io.WriteString(part, body); err != nil {
		_ = part.Close()
		return err
	}
	return part.Close()
}

func writeOutgoingAttachment(writer *gomail.Writer, attachment mail.OutgoingAttachment) error {
	reader, err := attachment.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	var header message.Header
	header.SetContentType(attachment.ContentType, nil)
	attachmentHeader := gomail.AttachmentHeader{Header: header}
	attachmentHeader.SetFilename(attachment.Filename)
	part, err := writer.CreateAttachment(attachmentHeader)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, reader); err != nil {
		_ = part.Close()
		return err
	}
	return part.Close()
}
