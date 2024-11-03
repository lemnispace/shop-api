package utils

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

func ProxyEventToHTTPRequest(req events.APIGatewayProxyRequest) (*http.Request, error) {
	body := req.Body
	if req.IsBase64Encoded {
		decoded, err := decodeBase64(body)
		if err != nil {
			return nil, err
		}
		body = decoded
	}

	httpRequest, err := http.NewRequest(req.HTTPMethod, req.Path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	for key, value := range req.Headers {
		httpRequest.Header.Set(key, value)
	}

	// Add query parameters
	q := httpRequest.URL.Query()
	for key, values := range req.QueryStringParameters {
		q.Set(key, values)
	}
	httpRequest.URL.RawQuery = q.Encode()

	return httpRequest, nil
}

func NewErrorResponse(statusCode int, message string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"error":"` + message + `"}`,
	}
}

type ResponseRecorder struct {
	Headers     http.Header
	Body        bytes.Buffer
	StatusCode  int
	wroteHeader bool
}

func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		Headers:    make(http.Header),
		StatusCode: http.StatusOK,
	}
}

func (rw *ResponseRecorder) Header() http.Header {
	return rw.Headers
}

func (rw *ResponseRecorder) Write(data []byte) (int, error) {
	return rw.Body.Write(data)
}

func (rw *ResponseRecorder) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		return
	}
	rw.StatusCode = statusCode
	rw.wroteHeader = true
}

func (rw *ResponseRecorder) GetProxyResponse() (events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{
		StatusCode:      rw.StatusCode,
		Headers:         make(map[string]string),
		Body:            rw.Body.String(),
		IsBase64Encoded: false,
	}
	for k, v := range rw.Headers {
		resp.Headers[k] = strings.Join(v, ",")
	}
	return resp, nil
}

// Helper function to decode base64 encoded strings
func decodeBase64(s string) (string, error) {
	data, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(s)))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
