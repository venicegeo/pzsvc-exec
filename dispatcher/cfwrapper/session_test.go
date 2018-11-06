package cfwrapper

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	cfclient "github.com/venicegeo/go-cfclient"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// plainHandler is a plain http.Handler for returning static data and counting hits
type plainHandler struct {
	path        string
	status      int
	data        []byte
	calledCount *int
}

func newPlainHandler(path string, status int, data []byte) *plainHandler {
	return &plainHandler{path: path, status: status, data: data, calledCount: new(int)}
}

func (h plainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.EscapedPath() == h.path {
		*h.calledCount++
		w.WriteHeader(h.status)
		w.Write(h.data)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// mockCFHandler is a self-bootstrapping wrapper to mock the CF API
type mockCFHandler struct {
	mockServer       *httptest.Server
	cfClient         *cfclient.Client
	wrappedHandler   http.Handler
	tokenHandlerFunc http.HandlerFunc
}

func setupMockCFHandler(wrapped http.Handler) (*mockCFHandler, error) {
	h := mockCFHandler{
		wrappedHandler: wrapped,
	}
	h.mockServer = httptest.NewServer(&h)
	cfClient, err := cfclient.NewClient(&cfclient.Config{
		ApiAddress: h.mockServer.URL,
		HttpClient: h.mockServer.Client(),
		Username:   "testuser",
		Password:   "testpassword",
	})
	if err != nil {
		return nil, err
	}
	h.cfClient = cfClient
	return &h, nil
}

const mockInfoResponse = `{"name":"","build":"","support":"https://support.pivotal.io",
  "version":0,"description":"","authorization_endpoint":"++MOCK_HOST++",
  "token_endpoint":"++MOCK_HOST++","min_cli_version":"6.23.0",
  "min_recommended_cli_version":"6.23.0","api_version":"2.75.0","app_ssh_endpoint":"ssh.example.localdomain:2222",
  "app_ssh_host_key_fingerprint":"00:00:00:00:00:00:00:ff:ff:ff:ff:ff:ff:be:ef:ed",
  "app_ssh_oauth_client":"ssh-proxy","routing_endpoint":"https://api.example.localdomain/routing",
  "logging_endpoint":"wss://logs.example.localdomain:4443",
  "doppler_logging_endpoint":"wss://doppler.example.localdomain:4443"
}`

func (h mockCFHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.EscapedPath() == "/v2/info" {
		w.WriteHeader(http.StatusOK)
		responseStr := strings.Replace(mockInfoResponse, "++MOCK_HOST++", h.mockServer.URL, -1)
		w.Write([]byte(responseStr))
		return
	}

	if r.URL.EscapedPath() == "/oauth/token" {
		if h.tokenHandlerFunc != nil {
			h.tokenHandlerFunc(w, r)
		} else {
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			responseStr := `{"access_token":"test-access-token",
				"token_type":"bearer",
				"expires_in":3600,
				"refresh_token":"test-refresh-token",
				"scope":"create"}`
			w.Write([]byte(responseStr))
		}
		return
	}

	h.wrappedHandler.ServeHTTP(w, r)
}
func (h mockCFHandler) TearDown() {
	h.mockServer.Close()
}

func TestWrappedCFSession_IsValid_Valid(t *testing.T) {
	// Setup
	h := newPlainHandler("/v2/apps", http.StatusOK, []byte(`{"total_results": 0, "total_pages": 1, "prev_url": null, "next_url": null, "resources": []}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: cfMock.cfClient}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.True(t, valid)
	assert.Nil(t, err)
}

func TestWrappedCFSession_IsValid_NilClient(t *testing.T) {
	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: nil}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.False(t, valid)
	assert.Nil(t, err)
}

func TestWrappedCFSession_IsValid_UnauthenticatedResponse(t *testing.T) {
	// Setup
	h := newPlainHandler("/v2/apps", http.StatusUnauthorized, []byte(`{"code": 10002, "error_code": "CF-NotAuthenticated", "description": "..."}}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: cfMock.cfClient}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.Nil(t, err)
	assert.False(t, valid)
	assert.Equal(t, 1, *h.calledCount)
}

func TestWrappedCFSession_IsValid_UnauthorizedResponse(t *testing.T) {
	// Setup
	h := newPlainHandler("/v2/apps", http.StatusForbidden, []byte(`{"code": 10003, "error_code": "CF-NotAuthorized", "description": "..."}}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: cfMock.cfClient}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.Nil(t, err)
	assert.False(t, valid)
	assert.Equal(t, 1, *h.calledCount)
}

func TestWrappedCFSession_IsValid_UnknownError(t *testing.T) {
	// Setup
	h := newPlainHandler("/v2/apps", http.StatusTeapot, []byte(`{"code": 10418, "error_code": "CF-Teapot", "description": "..."}}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: cfMock.cfClient}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.NotNil(t, err)
	assert.False(t, valid)
	assert.Equal(t, 1, *h.calledCount)
}

type expiredOauthRoundTripper struct {
	calledCount int
}

func (rt *expiredOauthRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.calledCount++
	return nil, errors.New("oauth2: cannot fetch token: 401 Unauthorized")
}
func TestWrappedCFSession_IsValid_OAuthExpiredRefreshToken(t *testing.T) {
	// Setup
	h := newPlainHandler("/v2/apps", http.StatusOK, []byte(`{"total_results": 0, "total_pages": 1, "prev_url": null, "next_url": null, "resources": []}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	brokenRoundTripper := &expiredOauthRoundTripper{}
	cfMock.cfClient.Config.HttpClient.Transport = brokenRoundTripper

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: cfMock.cfClient}
	valid, err := cfSession.IsValid()

	// Asserts
	assert.Nil(t, err)     // XXX: broken! (TDD)
	assert.False(t, valid) // XXX: broken! (TDD)
	assert.Equal(t, 1, brokenRoundTripper.calledCount)
	assert.Equal(t, 0, *h.calledCount)
}

func TestWrappedCFSession_CountTasksForApp_Success(t *testing.T) {
	// Setup
	h := newPlainHandler("/v3/apps/test-app-id/tasks", http.StatusOK, []byte(`{"pagination":{"total_results":3,"total_pages":1,"first":null,"last":null,"next":null,"previous": null},"resources":[{}, {}, {}]}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: cfMock.cfClient}
	count, err := cfSession.CountTasksForApp("test-app-id")

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, 3, count)
	assert.Equal(t, 1, *h.calledCount)
}

func TestWrappedCFSession_CountTasksForApp_Error(t *testing.T) {
	// Setup
	h := newPlainHandler("/v3/apps/test-app-id/tasks", http.StatusTeapot, []byte(`{"code": 10418, "error_code": "CF-Teapot", "description": "..."}}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: cfMock.cfClient}
	count, err := cfSession.CountTasksForApp("test-app-id")

	// Asserts
	assert.NotNil(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, 1, *h.calledCount)
}

func TestWrappedCFSession_CreateTask_Success(t *testing.T) {
	// Setup
	h := newPlainHandler("/v3/apps/test-app-id/tasks", http.StatusOK, []byte(`{}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: cfMock.cfClient}
	err = cfSession.CreateTask(TaskRequest{DropletGUID: "test-app-id"})

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, 1, *h.calledCount)
}

func TestWrappedCFSession_CreateTask_Error(t *testing.T) {
	// Setup
	h := newPlainHandler("/v3/apps/test-app-id/tasks", http.StatusTeapot, []byte(`{"code": 10418, "error_code": "CF-Teapot", "description": "..."}}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	// Tested code
	cfSession := wrappedCFSession{PzSession: &pzsvc.Session{}, Client: cfMock.cfClient}
	err = cfSession.CreateTask(TaskRequest{DropletGUID: "test-app-id"})

	// Asserts
	assert.NotNil(t, err)
	assert.Equal(t, 1, *h.calledCount)
}

func TestNewWrappedCFSession_Success(t *testing.T) {
	// Setup
	h := newPlainHandler("/", http.StatusOK, []byte(`{"total_results": 0, "total_pages": 1, "prev_url": null, "next_url": null, "resources": []}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	config := &FactoryConfig{
		APIAddress:  cfMock.cfClient.Config.ApiAddress,
		Username:    cfMock.cfClient.Config.Username,
		Password:    cfMock.cfClient.Config.Password,
		HTTPClient:  cfMock.cfClient.Config.HttpClient,
		Token:       cfMock.cfClient.Config.Token,
		TokenSource: cfMock.cfClient.Config.TokenSource,
	}

	// Tested code
	session, err := newWrappedCFSession(&pzsvc.Session{}, config)

	// Asserts
	assert.Nil(t, err)
	assert.NotNil(t, session)
}

func TestNewWrappedCFSession_Error(t *testing.T) {
	// Setup
	h := newPlainHandler("/", http.StatusInternalServerError, []byte(`{}`))
	cfMock, err := setupMockCFHandler(h)
	if err != nil {
		panic(err)
	}
	defer cfMock.TearDown()

	config := &FactoryConfig{
		APIAddress:  "bogus-address",
		Username:    cfMock.cfClient.Config.Username,
		Password:    cfMock.cfClient.Config.Password,
		HTTPClient:  cfMock.cfClient.Config.HttpClient,
		Token:       cfMock.cfClient.Config.Token,
		TokenSource: cfMock.cfClient.Config.TokenSource,
	}

	// Tested code
	session, err := newWrappedCFSession(&pzsvc.Session{}, config)

	// Asserts
	assert.Nil(t, session)
	assert.NotNil(t, err)
}
