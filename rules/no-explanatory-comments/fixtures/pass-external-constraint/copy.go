package storage

import (
	"encoding/xml"
	"fmt"
)

// S3 can answer a CopyObject with HTTP 200 and the failure in the body, so a
// 200 is not success until the body has been parsed for an error element.
func CheckCopy(status int, body []byte) error {
	if status != 200 {
		return fmt.Errorf("copy failed: status %d", status)
	}
	var failure struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	}
	if err := xml.Unmarshal(body, &failure); err == nil && failure.Code != "" {
		return fmt.Errorf("copy failed: %s: %s", failure.Code, failure.Message)
	}
	return nil
}
