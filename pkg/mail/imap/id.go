package imap

import (
	"fmt"
	"strconv"
	"strings"

	goimap "github.com/emersion/go-imap/v2"
)

const idSeparator = "\x1f"

func encodeMessageID(mailbox string, uid goimap.UID) string {
	return mailbox + idSeparator + strconv.FormatUint(uint64(uid), 10)
}

func decodeMessageID(id string) (string, goimap.UID, error) {
	if strings.Count(id, idSeparator) != 1 {
		return "", 0, fmt.Errorf("invalid IMAP message ID")
	}
	parts := strings.SplitN(id, idSeparator, 2)
	if parts[0] == "" || parts[1] == "" {
		return "", 0, fmt.Errorf("invalid IMAP message ID")
	}
	n, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil || n == 0 {
		return "", 0, fmt.Errorf("invalid IMAP message UID")
	}
	return parts[0], goimap.UID(n), nil
}

func parsePartID(partID string) ([]int, error) {
	if partID == "" {
		return nil, fmt.Errorf("part ID is required")
	}
	segments := strings.Split(partID, ".")
	part := make([]int, len(segments))
	for i, segment := range segments {
		n, err := strconv.Atoi(segment)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid part ID %q", partID)
		}
		part[i] = n
	}
	return part, nil
}

func formatPartID(part []int) string {
	segments := make([]string, len(part))
	for i, n := range part {
		segments[i] = strconv.Itoa(n)
	}
	return strings.Join(segments, ".")
}
