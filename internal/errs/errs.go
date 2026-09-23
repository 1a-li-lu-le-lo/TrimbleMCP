// Package errs defines the normalized error model shared by adapters, the MCP
// gateway, and the CLI. Raw upstream details never leave this package's
// Error.cause field; callers render only the safe fields.
package errs

import (
	"errors"
	"fmt"
)

// Code is a normalized, stable error code.
type Code string

const (
	Configuration          Code = "configuration_error"
	Authentication         Code = "authentication_error"
	Authorization          Code = "authorization_error"
	Entitlement            Code = "entitlement_error"
	TenantNotFound         Code = "tenant_not_found"
	OrganizationNotFound   Code = "organization_not_found"
	ProjectNotFound        Code = "project_not_found"
	ResourceNotFound       Code = "resource_not_found"
	Validation             Code = "validation_error"
	UnsupportedCapability  Code = "unsupported_capability"
	UnsupportedProduct     Code = "unsupported_product"
	CoordinateSystemNeeded Code = "coordinate_system_required"
	CoordinateTransform    Code = "coordinate_transformation_error"
	Unit                   Code = "unit_error"
	VersionConflict        Code = "version_conflict"
	FileTooLarge           Code = "file_too_large"
	FileRejected           Code = "file_rejected"
	RateLimited            Code = "rate_limited"
	UpstreamUnavailable    Code = "upstream_unavailable"
	UpstreamTimeout        Code = "upstream_timeout"
	UpstreamMalformed      Code = "upstream_malformed_response"
	UncertainMutation      Code = "uncertain_mutation"
	JobFailed              Code = "job_failed"
	PolicyDenied           Code = "policy_denied"
	ApprovalRequired       Code = "approval_required"
	Internal               Code = "internal_error"
)

var defaults = map[Code]struct {
	msg         string
	retryable   bool
	remediation string
}{
	Configuration:          {"The server is not configured for this operation.", false, "Ask an operator to check the Trimble MCP Bridge configuration."},
	Authentication:         {"Authentication with the upstream service failed.", false, "Re-authenticate through the configured Trimble Identity flow; never paste credentials into chat."},
	Authorization:          {"The caller is not authorized for this operation or resource.", false, "Request the required scope or project grant from an administrator."},
	Entitlement:            {"The account is not entitled to this product or capability.", false, "Confirm the Trimble subscription or licence for this product."},
	TenantNotFound:         {"The tenant is unknown.", false, "Check the configured tenant reference."},
	OrganizationNotFound:   {"The organization was not found.", false, "Resolve the organization with a list tool instead of guessing its ID."},
	ProjectNotFound:        {"The project was not found or is not visible to the caller.", false, "Resolve the project with trimble_list_projects instead of guessing its ID."},
	ResourceNotFound:       {"The resource was not found or is not visible to the caller.", false, "Resolve the resource with a list tool instead of guessing its ID."},
	Validation:             {"The request is invalid.", false, "Correct the highlighted input and retry."},
	UnsupportedCapability:  {"No configured adapter supports this capability.", false, "Call trimble_get_capabilities to see supported operations."},
	UnsupportedProduct:     {"The requested Trimble product is not configured.", false, "Call trimble_get_capabilities to see configured products."},
	CoordinateSystemNeeded: {"An explicit coordinate reference system is required.", false, "Provide the source (and target) CRS explicitly; it is never inferred."},
	CoordinateTransform:    {"The coordinate transformation could not be verified.", false, "Use a verified transformation with explicit source and target CRS."},
	Unit:                   {"Units are missing or inconsistent.", false, "Provide explicit units."},
	VersionConflict:        {"The resource changed since it was last read.", false, "Re-read the resource and prepare a new plan."},
	FileTooLarge:           {"The file exceeds the configured size limit.", false, "Use a smaller file or ask an operator about limits."},
	FileRejected:           {"The file was rejected by validation.", false, "Check the file type and contents."},
	RateLimited:            {"The upstream rate limit was reached.", true, "Wait for the indicated interval before retrying."},
	UpstreamUnavailable:    {"The upstream Trimble service is unavailable.", true, "Retry later; check trimble_get_capabilities for health."},
	UpstreamTimeout:        {"The upstream Trimble service timed out.", true, "Retry later with a narrower request."},
	UpstreamMalformed:      {"The upstream service returned a response that did not match its documented contract.", false, "Report this to an operator; the adapter may need re-verification."},
	UncertainMutation:      {"The outcome of a change is uncertain.", false, "Do not repeat the change; reconcile the resource state first."},
	JobFailed:              {"The job failed.", false, "Inspect the job record."},
	PolicyDenied:           {"Server policy denies this operation.", false, "This operation is not permitted for agents."},
	ApprovalRequired:       {"This operation requires explicit approval.", false, "Obtain a server-issued approval reference and retry."},
	Internal:               {"An internal error occurred.", true, "Retry; if it persists, report the request ID to an operator."},
}

// Error is a normalized error. Message is safe to show to agents and users.
type Error struct {
	Code        Code   `json:"code"`
	Message     string `json:"safe_message"`
	Retryable   bool   `json:"retryable"`
	Remediation string `json:"remediation"`
	RetryAfter  int    `json:"retry_after_seconds,omitempty"`
	cause       error
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

// Unwrap exposes the private cause for server-side logging and errors.Is.
func (e *Error) Unwrap() error { return e.cause }

// New builds an Error with the default safe message for code.
func New(code Code) *Error {
	d, ok := defaults[code]
	if !ok {
		d = defaults[Internal]
	}
	return &Error{Code: code, Message: d.msg, Retryable: d.retryable, Remediation: d.remediation}
}

// Newf builds an Error with a caller-provided safe message. The message must
// never contain secrets or raw upstream payloads.
func Newf(code Code, format string, args ...any) *Error {
	e := New(code)
	e.Message = fmt.Sprintf(format, args...)
	return e
}

// WithSafeMessage replaces the client-visible message.
func (e *Error) WithSafeMessage(msg string) *Error {
	e.Message = msg
	return e
}

// Wrap attaches a private cause that is not rendered to clients.
func Wrap(code Code, cause error) *Error {
	e := New(code)
	e.cause = cause
	return e
}

// WithCause returns e with a private cause attached.
func (e *Error) WithCause(cause error) *Error {
	e.cause = cause
	return e
}

// As normalizes any error into an *Error, mapping unknown errors to Internal.
func As(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return Wrap(Internal, err)
}

// Is reports whether err carries code.
func Is(err error, code Code) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}
