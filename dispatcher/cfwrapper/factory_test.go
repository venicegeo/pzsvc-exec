package cfwrapper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

type mockSession struct {
	Valid    bool
	ValidErr error
}

func (s mockSession) IsValid() (bool, error) {
	return s.Valid, s.ValidErr
}

func (s mockSession) CountTasksForApp(appID string) (int, error) {
	return 0, nil
}

func (s mockSession) CreateTask(request TaskRequest) error {
	return nil
}

func TestNewFactory_Success(t *testing.T) {
	// Setup
	numCalls := 0
	pzSession := &pzsvc.Session{}
	mockSessionInstance := &mockSession{}
	factoryConfig := &FactoryConfig{createSessionFunc: func(pzSession *pzsvc.Session, config *FactoryConfig) (CFSession, error) {
		numCalls++
		return mockSessionInstance, nil
	}}

	// Tested code
	factory := NewFactory(pzSession, factoryConfig)

	// Asserts
	assert.NotNil(t, factory)
	assert.NotNil(t, factory.createSession)
	assert.Equal(t, factory.pzSession, pzSession)
	assert.Equal(t, 0, numCalls) // A session should not be immediately instantiated
}

func TestFactory_RefreshCachedSession_Success(t *testing.T) {
	// Setup
	numCalls := 0
	mockSessionInstance := &mockSession{}
	factory := NewFactory(&pzsvc.Session{}, &FactoryConfig{createSessionFunc: func(pzSession *pzsvc.Session, config *FactoryConfig) (CFSession, error) {
		numCalls++
		return mockSessionInstance, nil
	}})

	// Tested code

	err := factory.RefreshCachedClient()

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, 1, numCalls)
	assert.IsType(t, &mockSession{}, factory.cachedSession)
	assert.Equal(t, mockSessionInstance, factory.cachedSession.(*mockSession))
}

func TestFactory_GetSession_SuccessCacheHit(t *testing.T) {
	// Setup
	numCalls := 0
	mockSessionInstance := &mockSession{Valid: true}
	factory := NewFactory(&pzsvc.Session{}, &FactoryConfig{createSessionFunc: func(pzSession *pzsvc.Session, config *FactoryConfig) (CFSession, error) {
		numCalls++
		return mockSessionInstance, nil
	}})
	factory.cachedSession = mockSessionInstance

	// Tested code

	session, err := factory.GetSession()

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, 0, numCalls) // Should not be called because the cached session is valid
	assert.Equal(t, mockSessionInstance, session)
}

func TestFactory_GetSession_SuccessCacheMissNotNil(t *testing.T) {
	// Setup
	numCalls := 0
	invalidMockSessionInstance := &mockSession{Valid: false}
	validMockSessionInstance := &mockSession{Valid: true}
	factory := NewFactory(&pzsvc.Session{}, &FactoryConfig{createSessionFunc: func(pzSession *pzsvc.Session, config *FactoryConfig) (CFSession, error) {
		numCalls++
		return validMockSessionInstance, nil
	}})
	factory.cachedSession = invalidMockSessionInstance

	// Tested code

	session, err := factory.GetSession()

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, 1, numCalls)
	assert.Equal(t, validMockSessionInstance, session)
}

func TestFactory_GetSession_SuccessCacheMissNil(t *testing.T) {
	// Setup
	numCalls := 0
	validMockSessionInstance := &mockSession{Valid: true}
	factory := NewFactory(&pzsvc.Session{}, &FactoryConfig{createSessionFunc: func(pzSession *pzsvc.Session, config *FactoryConfig) (CFSession, error) {
		numCalls++
		return validMockSessionInstance, nil
	}})
	factory.cachedSession = nil

	// Tested code

	session, err := factory.GetSession()

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, 1, numCalls)
	assert.Equal(t, validMockSessionInstance, session)
}

func TestFactory_GetSession_ErrorDuringValidation(t *testing.T) {
	// Setup
	numCalls := 0
	invalidMockSessionInstance := &mockSession{Valid: false, ValidErr: errors.New("test error 1")}
	factory := NewFactory(&pzsvc.Session{}, &FactoryConfig{createSessionFunc: func(pzSession *pzsvc.Session, config *FactoryConfig) (CFSession, error) {
		numCalls++
		return nil, errors.New("test error 2")
	}})
	factory.cachedSession = invalidMockSessionInstance

	// Tested code

	session, err := factory.GetSession()

	// Asserts
	assert.Equal(t, 0, numCalls) // Should fail during validation, before trying to make a new session
	assert.Nil(t, session)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test error 1")
}

func TestFactory_GetSession_ErrorDuringCreation(t *testing.T) {
	// Setup
	numCalls := 0
	invalidMockSessionInstance := &mockSession{Valid: false}
	factory := NewFactory(&pzsvc.Session{}, &FactoryConfig{createSessionFunc: func(pzSession *pzsvc.Session, config *FactoryConfig) (CFSession, error) {
		numCalls++
		return nil, errors.New("test error 2")
	}})
	factory.cachedSession = invalidMockSessionInstance

	// Tested code

	session, err := factory.GetSession()

	// Asserts
	assert.Equal(t, 1, numCalls)
	assert.Nil(t, session)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test error 2")
}
