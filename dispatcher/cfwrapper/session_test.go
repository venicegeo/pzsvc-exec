package cfwrapper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	cfclient "github.com/venicegeo/go-cfclient"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"golang.org/x/oauth2"
)

type mockCFHandler struct {
	ResponseStatus int
	ResponseBytes  []byte
	AssertRequest  func(*http.Request)
}

const mockInfoResponse = `{"name":"","build":"","support":"https://support.pivotal.io",
  "version":0,"description":"","authorization_endpoint":"https://authorization.example.localdomain",
  "token_endpoint":"https://token.example.localdomain","min_cli_version":"6.23.0",
  "min_recommended_cli_version":"6.23.0","api_version":"2.75.0","app_ssh_endpoint":"ssh.example.localdomain:2222",
  "app_ssh_host_key_fingerprint":"00:00:00:00:00:00:00:ff:ff:ff:ff:ff:ff:be:ef:ed",
  "app_ssh_oauth_client":"ssh-proxy","routing_endpoint":"https://api.example.localdomain/routing",
  "logging_endpoint":"wss://logs.example.localdomain:4443",
  "doppler_logging_endpoint":"wss://doppler.example.localdomain:4443"
}`

func (h mockCFHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.EscapedPath() == "/v2/info" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockInfoResponse))
		return
	}
	if h.AssertRequest != nil {
		h.AssertRequest(r)
	}
	w.WriteHeader(h.ResponseStatus)
	w.Write(h.ResponseBytes)
}

type mockTokenSource struct{}

const mockOauth2Token = "mock-oauth2-token"

func createMockCFClient(h http.Handler) (*httptest.Server, *cfclient.Client, error) {
	server := httptest.NewServer(h)
	client, err := cfclient.NewClient(&cfclient.Config{ApiAddress: server.URL, HttpClient: server.Client(), Token: mockOauth2Token, TokenSource: mockTokenSource{}})
	return server, client, err
}

func (ts mockTokenSource) Token() (*oauth2.Token, error) {
	return nil, nil
}

func TestWrappedCFSession_IsValid_Valid(t *testing.T) {
	// Setup
	server, client, _ := createMockCFClient(mockCFHandler{
		ResponseStatus: http.StatusOK,
		ResponseBytes:  []byte(`{"total_results": 0, "total_pages": 1, "prev_url": null, "next_url": null, "resources": []}`)},
	)
	defer server.Close()

	// Tested code
	cfSession := wrappedCFSession{Client: client}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.True(t, valid)
	assert.Nil(t, err)
}

func TestWrappedCFSession_IsValid_NilClient(t *testing.T) {
	// Tested code
	cfSession := wrappedCFSession{Client: nil}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.False(t, valid)
	assert.Nil(t, err)
}

func TestWrappedCFSession_IsValid_UnauthenticatedResponse(t *testing.T) {
	// Setup
	numHTTPQueries := 0
	handler := &mockCFHandler{
		ResponseStatus: http.StatusUnauthorized, // 401
		ResponseBytes:  []byte(`{"code": 10002, "error_code": "CF-NotAuthenticated", "description": "..."}}`),
		AssertRequest: func(r *http.Request) {
			if r.URL.EscapedPath() == "/v2/apps" {
				numHTTPQueries++
			}
		},
	}
	server, client, _ := createMockCFClient(*handler)
	defer server.Close()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: client}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.Equal(t, 1, numHTTPQueries)
	assert.False(t, valid)
	assert.Nil(t, err)
}

func TestWrappedCFSession_IsValid_UnauthorizedResponse(t *testing.T) {
	// Setup
	numHTTPQueries := 0
	handler := &mockCFHandler{
		ResponseStatus: http.StatusForbidden, // 403
		ResponseBytes:  []byte(`{"code": 10003, "error_code": "CF-NotAuthorized", "description": "..."}}`),
		AssertRequest: func(r *http.Request) {
			if r.URL.EscapedPath() == "/v2/apps" {
				numHTTPQueries++
			}
		},
	}
	server, client, _ := createMockCFClient(*handler)
	defer server.Close()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: client}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.Equal(t, 1, numHTTPQueries)
	assert.False(t, valid)
	assert.Nil(t, err)
}

func TestWrappedCFSession_IsValid_UnknownError(t *testing.T) {
	// Setup
	numHTTPQueries := 0
	handler := &mockCFHandler{
		ResponseStatus: http.StatusTeapot, // 418
		ResponseBytes:  []byte(`{"code": 10418, "error_code": "CF-Teapot", "description": "..."}}`),
		AssertRequest: func(r *http.Request) {
			if r.URL.EscapedPath() == "/v2/apps" {
				numHTTPQueries++
			}
		},
	}
	server, client, _ := createMockCFClient(*handler)
	defer server.Close()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: client}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.Equal(t, 1, numHTTPQueries)
	assert.False(t, valid)
	assert.NotNil(t, err)
}

func TestWrappedCFSession_CountTasksForApp_Success(t *testing.T) {
	// Setup
	numHTTPQueries := 0
	handler := &mockCFHandler{
		ResponseStatus: http.StatusOK, // 200
		ResponseBytes:  []byte(`{"pagination":{"total_results":3,"total_pages":1,"first":null,"last":null,"next":null,"previous": null},"resources":[{}, {}, {}]}`),
		AssertRequest: func(r *http.Request) {
			if r.URL.EscapedPath() == "/v3/apps/test-app-id/tasks" {
				numHTTPQueries++
			}
		},
	}
	server, client, _ := createMockCFClient(*handler)
	defer server.Close()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: client}
	count, err := cfSession.CountTasksForApp("test-app-id")

	// Asserts
	assert.Equal(t, 1, numHTTPQueries)
	assert.Equal(t, 3, count)
	assert.Nil(t, err)
}

func TestWrappedCFSession_CountTasksForApp_Error(t *testing.T) {
	// Setup
	numHTTPQueries := 0
	handler := &mockCFHandler{
		ResponseStatus: http.StatusTeapot, // 418
		ResponseBytes:  []byte(`{"code": 10418, "error_code": "CF-Teapot", "description": "..."}}`),
		AssertRequest: func(r *http.Request) {
			if r.URL.EscapedPath() == "/v3/apps/test-app-id/tasks" {
				numHTTPQueries++
			}
		},
	}
	server, client, _ := createMockCFClient(*handler)
	defer server.Close()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: client}
	count, err := cfSession.CountTasksForApp("test-app-id")

	// Asserts
	assert.Equal(t, 1, numHTTPQueries)
	assert.Equal(t, 0, count)
	assert.NotNil(t, err)
}

func TestWrappedCFSession_CreateTask_Success(t *testing.T) {
	// Setup
	numHTTPQueries := 0
	handler := &mockCFHandler{
		ResponseStatus: http.StatusOK, // 200
		ResponseBytes:  []byte(`{}`),
		AssertRequest: func(r *http.Request) {
			if r.URL.EscapedPath() == "/v3/apps/test-app-id/tasks" && r.Method == "POST" {
				numHTTPQueries++
			}
		},
	}
	server, client, _ := createMockCFClient(*handler)
	defer server.Close()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: client}
	err := cfSession.CreateTask(TaskRequest{DropletGUID: "test-app-id"})

	// Asserts
	assert.Equal(t, 1, numHTTPQueries)
	assert.Nil(t, err)
}

func TestWrappedCFSession_CreateTask_Error(t *testing.T) {
	// Setup
	numHTTPQueries := 0
	handler := &mockCFHandler{
		ResponseStatus: http.StatusTeapot, // 418
		ResponseBytes:  []byte(`{"code": 10418, "error_code": "CF-Teapot", "description": "..."}}`),
		AssertRequest: func(r *http.Request) {
			if r.URL.EscapedPath() == "/v3/apps/test-app-id/tasks" && r.Method == "POST" {
				numHTTPQueries++
			}
		},
	}
	server, client, _ := createMockCFClient(*handler)
	defer server.Close()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: client}
	err := cfSession.CreateTask(TaskRequest{DropletGUID: "test-app-id"})

	// Asserts
	assert.Equal(t, 1, numHTTPQueries)
	assert.NotNil(t, err)
}

func TestNewWrappedCFSession_Success(t *testing.T) {
	// Setup
	server, client, _ := createMockCFClient(mockCFHandler{
		ResponseStatus: http.StatusOK,
		ResponseBytes:  []byte(`{"total_results": 0, "total_pages": 1, "prev_url": null, "next_url": null, "resources": []}`)},
	)
	defer server.Close()
	pzSession := &pzsvc.Session{}
	config := &FactoryConfig{
		APIAddress:  client.Config.ApiAddress,
		Username:    client.Config.Username,
		Password:    client.Config.Password,
		HTTPClient:  client.Config.HttpClient,
		Token:       client.Config.Token,
		TokenSource: client.Config.TokenSource,
	}

	// Tested code
	session, err := newWrappedCFSession(pzSession, config)

	// Asserts
	assert.NotNil(t, session)
	assert.Nil(t, err)
}

func TestNewWrappedCFSession_Error(t *testing.T) {
	// Setup
	server, client, _ := createMockCFClient(mockCFHandler{})
	pzSession := &pzsvc.Session{}
	config := FactoryConfig{
		APIAddress:  client.Config.ApiAddress,
		Username:    client.Config.Username,
		Password:    client.Config.Password,
		HTTPClient:  client.Config.HttpClient,
		Token:       client.Config.Token,
		TokenSource: client.Config.TokenSource,
	}
	server.Close() // Force connection failure by shutting down the server early

	// Tested code
	session, err := newWrappedCFSession(pzSession, &config)

	// Asserts
	assert.Nil(t, session)
	assert.NotNil(t, err)
}
