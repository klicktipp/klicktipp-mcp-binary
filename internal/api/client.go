package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/klicktipp/klicktipp-binary-mcp/internal/auth"
	"github.com/klicktipp/klicktipp-binary-mcp/internal/config"
	kerrors "github.com/klicktipp/klicktipp-binary-mcp/internal/errors"
)

type Client struct {
	cfg        config.Config
	httpClient *http.Client
	session    *sessionAuth
}

type sessionAuth struct {
	SessID      string `json:"sessid"`
	SessionName string `json:"session_name"`
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) ListTags(ctx context.Context) (map[string]string, error) {
	var result map[string]string
	err := c.request(ctx, http.MethodGet, "/tag", nil, nil, &result, true)
	return result, err
}

func (c *Client) GetTag(ctx context.Context, tagID any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodGet, "/tag/"+url.PathEscape(fmt.Sprint(tagID)), nil, nil, &result, true)
	return result, err
}

func (c *Client) ListFields(ctx context.Context) (map[string]string, error) {
	var result map[string]string
	err := c.request(ctx, http.MethodGet, "/field", nil, nil, &result, true)
	return result, err
}

func (c *Client) GetField(ctx context.Context, fieldID string) (any, error) {
	normalized := normalizeFieldID(fieldID)
	var result any
	err := c.request(ctx, http.MethodGet, "/field/"+url.PathEscape(normalized), nil, nil, &result, true)
	return result, err
}

func (c *Client) ListOptInProcesses(ctx context.Context) (map[string]string, error) {
	var result map[string]string
	err := c.request(ctx, http.MethodGet, "/list", nil, nil, &result, true)
	return result, err
}

func (c *Client) GetOptInProcess(ctx context.Context, listID any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodGet, "/list/"+url.PathEscape(fmt.Sprint(listID)), nil, nil, &result, true)
	return result, err
}

func (c *Client) GetOptInProcessRedirect(ctx context.Context, payload map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodPost, "/list/redirect", payload, nil, &result, true)
	return result, err
}

func (c *Client) ListContacts(ctx context.Context, query map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodGet, "/subscriber", nil, query, &result, true)
	return result, err
}

func (c *Client) GetContact(ctx context.Context, subscriberID any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodGet, "/subscriber/"+url.PathEscape(fmt.Sprint(subscriberID)), nil, nil, &result, true)
	return result, err
}

func (c *Client) SearchContact(ctx context.Context, payload map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodPost, "/subscriber/search", payload, nil, &result, true)
	return result, err
}

func (c *Client) SearchTaggedContacts(ctx context.Context, payload map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodPost, "/subscriber/tagged", payload, nil, &result, true)
	return result, err
}

func (c *Client) CreateTag(ctx context.Context, payload map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodPost, "/tag", payload, nil, &result, true)
	return result, err
}

func (c *Client) UpdateTag(ctx context.Context, tagID any, payload map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodPut, "/tag/"+url.PathEscape(fmt.Sprint(tagID)), payload, nil, &result, true)
	return result, err
}

func (c *Client) DeleteTag(ctx context.Context, tagID any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodDelete, "/tag/"+url.PathEscape(fmt.Sprint(tagID)), nil, nil, &result, true)
	return result, err
}

func (c *Client) CreateOrUpdateContact(ctx context.Context, payload map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodPost, "/subscriber", payload, nil, &result, true)
	return result, err
}

func (c *Client) UpdateContact(ctx context.Context, subscriberID any, payload map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodPut, "/subscriber/"+url.PathEscape(fmt.Sprint(subscriberID)), payload, nil, &result, true)
	return result, err
}

func (c *Client) DeleteContact(ctx context.Context, subscriberID any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodDelete, "/subscriber/"+url.PathEscape(fmt.Sprint(subscriberID)), nil, nil, &result, true)
	return result, err
}

func (c *Client) UnsubscribeContact(ctx context.Context, payload map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodPost, "/subscriber/unsubscribe", payload, nil, &result, true)
	return result, err
}

func (c *Client) TagContact(ctx context.Context, payload map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodPost, "/subscriber/tag", payload, nil, &result, true)
	return result, err
}

func (c *Client) UntagContact(ctx context.Context, payload map[string]any) (any, error) {
	var result any
	err := c.request(ctx, http.MethodPost, "/subscriber/untag", payload, nil, &result, true)
	return result, err
}

func (c *Client) request(ctx context.Context, method string, path string, body any, query map[string]any, target any, retryOnAuthError bool) error {
	reqBody, err := marshalBody(body)
	if err != nil {
		return err
	}

	requestURL := c.cfg.BaseURL + path + buildQueryString(query)
	req, err := http.NewRequestWithContext(ctx, method, requestURL, reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if err := c.applyAuth(req); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	parsedBody := parseBody(responseBytes)
	if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && c.cfg.AuthMode == "session" && retryOnAuthError {
		c.session = nil
		return c.request(ctx, method, path, body, query, target, false)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &kerrors.APIError{
			Status:  resp.StatusCode,
			Body:    parsedBody,
			Message: toErrorMessage(resp.StatusCode, parsedBody),
		}
	}

	if target == nil {
		return nil
	}

	switch typed := target.(type) {
	case *any:
		*typed = parsedBody
		return nil
	default:
		raw, err := json.Marshal(parsedBody)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, target)
	}
}

func (c *Client) applyAuth(req *http.Request) error {
	if c.cfg.AuthMode == "partner" {
		headers, err := auth.PartnerHeaders(c.cfg)
		if err != nil {
			return err
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		return nil
	}

	if c.session == nil {
		session, err := c.login(req.Context())
		if err != nil {
			return err
		}
		c.session = session
	}

	req.Header.Set("X-Session-Id", c.session.SessID)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", c.session.SessionName, c.session.SessID))
	return nil
}

func (c *Client) login(ctx context.Context) (*sessionAuth, error) {
	username := strings.TrimSpace(c.cfg.Username)
	password := c.cfg.Password
	if username == "" || password == "" {
		return nil, fmt.Errorf("KT_USERNAME and KT_PASSWORD are required for session auth")
	}

	payload := map[string]string{
		"username": username,
		"password": password,
	}

	requestURL := c.cfg.BaseURL + "/account/login"
	reqBody, err := marshalBody(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	parsedBody := parseBody(responseBytes)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &kerrors.APIError{
			Status:  resp.StatusCode,
			Body:    parsedBody,
			Message: toErrorMessage(resp.StatusCode, parsedBody),
		}
	}

	raw, err := json.Marshal(parsedBody)
	if err != nil {
		return nil, err
	}

	var session sessionAuth
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, err
	}
	if session.SessID == "" || session.SessionName == "" {
		return nil, fmt.Errorf("login succeeded but KlickTipp did not return sessid and session_name")
	}

	return &session, nil
}

func marshalBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(encoded), nil
}

func parseBody(data []byte) any {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil
	}

	var parsed any
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return string(trimmed)
	}
	return parsed
}

func toErrorMessage(status int, body any) string {
	if body == nil {
		return fmt.Sprintf("KlickTipp API request failed with status %d.", status)
	}
	return fmt.Sprintf("KlickTipp API request failed with status %d: %v", status, body)
}

func buildQueryString(query map[string]any) string {
	if len(query) == 0 {
		return ""
	}

	params := url.Values{}
	for key, value := range query {
		if value == nil {
			continue
		}

		switch typed := value.(type) {
		case []string:
			if len(typed) > 0 {
				params.Set(key, strings.Join(typed, ","))
			}
		case []any:
			if len(typed) > 0 {
				values := make([]string, 0, len(typed))
				for _, item := range typed {
					values = append(values, fmt.Sprint(item))
				}
				params.Set(key, strings.Join(values, ","))
			}
		default:
			params.Set(key, fmt.Sprint(typed))
		}
	}

	encoded := params.Encode()
	if encoded == "" {
		return ""
	}
	return "?" + encoded
}

func normalizeFieldID(fieldID string) string {
	value := strings.TrimSpace(fieldID)
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "field") && len(value) > 5 {
		suffix := value[5:]
		if suffix != "" {
			return suffix
		}
	}
	return value
}
