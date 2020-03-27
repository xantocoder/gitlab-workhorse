package headers

import (
	"fmt"
	"net/http"
	"regexp"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/utils/svg"
)

var (
	ImageTypeRegex   = regexp.MustCompile(`^image/*`)
	SvgMimeTypeRegex = regexp.MustCompile(`^image/svg\+xml$`)

	TextTypeRegex = regexp.MustCompile(`^text/*`)

	VideoTypeRegex = regexp.MustCompile(`^video/*`)
	AudioTypeRegex = regexp.MustCompile(`^audio/*`)

	PdfTypeRegex = regexp.MustCompile(`application\/pdf`)

	AttachmentRegex = regexp.MustCompile(`^attachment`)
	InlineRegex     = regexp.MustCompile(`^inline`)
	MimeTypeRegex   = regexp.MustCompile(`^(.+)\/(.+)$`)
)

// Mime types that can't be inlined. Usually subtypes of main types
var forbiddenInlineTypes = []*regexp.Regexp{SvgMimeTypeRegex}

// Mime types that can be inlined. We can add global types like "image/" or
// specific types like "text/plain". If there is a specific type inside a global
// allowed type that can't be inlined we must add it to the forbiddenInlineTypes var.
// One example of this is the mime type "image". We allow all images to be
// inlined except for SVGs.
var allowedInlineTypes = []*regexp.Regexp{ImageTypeRegex, TextTypeRegex, VideoTypeRegex, AudioTypeRegex, PdfTypeRegex}

func SafeContentHeaders(data []byte, existingContentType string, existingContentDisposition string) (string, string) {
	contentType := safeContentType(data, existingContentType)
	contentDisposition := safeContentDisposition(contentType, existingContentDisposition)
	return contentType, contentDisposition
}

func safeContentType(data []byte, existingContentType string) string {
	// Special case for svg because DetectContentType detects it as text
	if SvgMimeTypeRegex.MatchString(existingContentType) || svg.Is(data) {
		return "image/svg+xml"
	}

	contentType := determineFinalContentType(existingContentType, http.DetectContentType(data))

	// If the content is text type, we set it to plain, because we don't
	// want to render it inline if they're html or javascript
	if isType(contentType, TextTypeRegex) {
		return "text/plain; charset=utf-8"
	}

	return contentType
}

// Here we check whether contentType and detectedContentType are too different from each other.
// Depending on that, we pass that the existing content type or the detected one
func determineFinalContentType(existingContentType string, detectedContentType string) string {
	// If existing Content-Type is blank we use the detected one
	if existingContentType == "" {
		return detectedContentType
	}

	// Get the first part of content type on both content types
	// if they're not the same we return the detected content type
	existingPrimaryType, err := primaryContentType(existingContentType)
	if err != nil {
		// If there is an error is because the Content Type
		// received from Rails is invalid. We returned the detected one
		return detectedContentType
	}

	currentPrimaryType, err := primaryContentType(detectedContentType)
	if err != nil {
		// Having an error here is bad. It's strange we cannot detect
		// the content type. For the sake of security, we return a
		// application/octect-stream contentType to return an
		// attachment Content-Disposition
		return "application/octet-stream"
	}

	// If the primary existing content type matches the detected
	// one, we return the existing content type from the response
	if existingPrimaryType == currentPrimaryType {
		return existingContentType
	}

	return detectedContentType
}

func safeContentDisposition(contentType string, contentDisposition string) string {
	// If the existing disposition is attachment we return that. This allow us
	// to force a download from GitLab (ie: RawController)
	if AttachmentRegex.MatchString(contentDisposition) {
		return contentDisposition
	}

	// Checks for mime types that are forbidden to be inline
	for _, element := range forbiddenInlineTypes {
		if isType(contentType, element) {
			return attachmentDisposition(contentDisposition)
		}
	}

	// Checks for mime types allowed to be inline
	for _, element := range allowedInlineTypes {
		if isType(contentType, element) {
			return inlineDisposition(contentDisposition)
		}
	}

	// Anything else is set to attachment
	return attachmentDisposition(contentDisposition)
}

func attachmentDisposition(contentDisposition string) string {
	if contentDisposition == "" {
		return "attachment"
	}

	if InlineRegex.MatchString(contentDisposition) {
		return InlineRegex.ReplaceAllString(contentDisposition, "attachment")
	}

	return contentDisposition
}

func inlineDisposition(contentDisposition string) string {
	if contentDisposition == "" {
		return "inline"
	}

	if AttachmentRegex.MatchString(contentDisposition) {
		return AttachmentRegex.ReplaceAllString(contentDisposition, "inline")
	}

	return contentDisposition
}

func isType(contentType string, mimeType *regexp.Regexp) bool {
	return mimeType.MatchString(contentType)
}

func primaryContentType(contentType string) (string, error) {
	result := MimeTypeRegex.FindStringSubmatch(contentType)

	if len(result) != 3 {
		return "", fmt.Errorf("invalid Content Type")
	}

	return result[1], nil
}
