package template

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

const (
	testPoolID        = "b7569d99-30f8-178a-7d94-801de3e29b5b"
	testTemplateID1   = "b7569d99-30f8-178a-7d94-801de3e29b5b-f873abe0-b138-4995-8f6f-498b423d234d"
	testTemplateID2   = "b7569d99-30f8-178a-7d94-801de3e29b5b-a3d70e4d-c5ac-4dfb-999b-30a0a7efe546"
	testTokenValue    = "test-token"
	testTemplateUUID1 = "f873abe0-b138-4995-8f6f-498b423d234d"
	// testTemplateIDNotFound is a well-formed composite id that the mock
	// server does not know about.
	testTemplateIDNotFound = testPoolID + "-00000000-0000-0000-0000-000000000000"
)

var mockTemplates = func() []*payloads.Template {
	return []*payloads.Template{
		{
			ID:              testTemplateID1,
			UUID:            uuid.Must(uuid.FromString(testTemplateUUID1)),
			Type:            payloads.ResourceTypeVMTemplate,
			Pool:            uuid.Must(uuid.FromString(testPoolID)),
			PoolID:          uuid.Must(uuid.FromString(testPoolID)),
			Container:       uuid.Must(uuid.FromString(testPoolID)),
			NameLabel:       "Oracle Linux 7",
			NameDescription: "Test template 1",
			PowerState:      "Halted",
			IsDefault:       true,
			Memory:          payloads.Memory{Size: 4294967296},
			CPUs:            payloads.CPUs{Number: 1, Max: 1},
			Tags:            []string{},
			TemplateInfo: payloads.TemplateInfo{
				Arch:           "x86_64",
				InstallMethods: []string{"cdrom", "http"},
				Disks: []payloads.TemplateDisk{
					{Bootable: true, Size: 10737418240, Type: "system"},
				},
			},
		},
		{
			ID:         testTemplateID2,
			UUID:       uuid.Must(uuid.FromString("a3d70e4d-c5ac-4dfb-999b-30a0a7efe546")),
			Type:       payloads.ResourceTypeVMTemplate,
			Pool:       uuid.Must(uuid.FromString(testPoolID)),
			PoolID:     uuid.Must(uuid.FromString(testPoolID)),
			Container:  uuid.Must(uuid.FromString(testPoolID)),
			NameLabel:  "CentOS Stream 9",
			PowerState: "Halted",
			Memory:     payloads.Memory{Size: 2147483648},
			CPUs:       payloads.CPUs{Number: 1, Max: 1},
		},
	}
}

func setupTestServer(t *testing.T) (*httptest.Server, *Service) {
	t.Helper()
	mux := http.NewServeMux()

	// GET /rest/v0/vm-templates - List all templates
	mux.HandleFunc("GET /rest/v0/vm-templates", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockTemplates()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// GET /rest/v0/vm-templates/{id} - Get specific template (composite id)
	mux.HandleFunc("GET /rest/v0/vm-templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		idStr := r.PathValue("id")

		var tmpl *payloads.Template
		switch idStr {
		case testTemplateID1:
			tmpl = mockTemplates()[0]
		case testTemplateID2:
			tmpl = mockTemplates()[1]
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(tmpl); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	server := httptest.NewServer(mux)

	restClient := &client.Client{
		HttpClient: server.Client(),
		BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
		AuthToken:  testTokenValue,
	}

	log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	return server, New(restClient, log).(*Service)
}

func TestGet(t *testing.T) {
	t.Run("get existing template by composite ID", func(t *testing.T) {
		server, svc := setupTestServer(t)
		defer server.Close()

		tmpl, err := svc.Get(context.Background(), testTemplateID1)
		require.NoError(t, err)
		require.NotNil(t, tmpl)
		assert.Equal(t, testTemplateID1, tmpl.ID)
		assert.Equal(t, testTemplateUUID1, tmpl.UUID.String())
		assert.Equal(t, "Oracle Linux 7", tmpl.NameLabel)
		assert.True(t, tmpl.IsDefault)
		assert.Equal(t, payloads.ResourceTypeVMTemplate, tmpl.Type)
		assert.Equal(t, testPoolID, tmpl.PoolID.String())
	})

	t.Run("get non-existent template by ID", func(t *testing.T) {
		server, svc := setupTestServer(t)
		defer server.Close()

		tmpl, err := svc.Get(context.Background(), testTemplateIDNotFound)
		assert.Error(t, err)
		assert.Nil(t, tmpl)
	})
}

func TestGetAll(t *testing.T) {
	t.Run("successfully retrieves all templates", func(t *testing.T) {
		server, svc := setupTestServer(t)
		defer server.Close()

		templates, err := svc.GetAll(context.Background(), 0, "")
		require.NoError(t, err)
		require.Len(t, templates, 2)
		assert.Equal(t, testTemplateID1, templates[0].ID)
		assert.Equal(t, testTemplateID2, templates[1].ID)
	})

	t.Run("passes limit and filter parameters", func(t *testing.T) {
		limit := 10
		filter := "name_label:Oracle"

		var receivedQuery url.Values
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedQuery = r.URL.Query()
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]*payloads.Template{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		server := httptest.NewServer(handler)
		defer server.Close()

		restClient := &client.Client{
			HttpClient: server.Client(),
			BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
			AuthToken:  testTokenValue,
		}
		log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
		require.NoError(t, err)
		svc := New(restClient, log).(*Service)

		_, err = svc.GetAll(context.Background(), limit, filter)
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf("%d", limit), receivedQuery.Get("limit"))
		assert.Equal(t, filter, receivedQuery.Get("filter"))
		assert.Equal(t, "*", receivedQuery.Get("fields"))
	})

	t.Run("does not send limit param when zero", func(t *testing.T) {
		var receivedQuery url.Values
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedQuery = r.URL.Query()
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]*payloads.Template{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		server := httptest.NewServer(handler)
		defer server.Close()

		restClient := &client.Client{
			HttpClient: server.Client(),
			BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
			AuthToken:  testTokenValue,
		}
		log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
		require.NoError(t, err)
		svc := New(restClient, log).(*Service)

		_, err = svc.GetAll(context.Background(), 0, "")
		require.NoError(t, err)

		assert.Empty(t, receivedQuery.Get("limit"))
		assert.Empty(t, receivedQuery.Get("filter"))
		assert.Equal(t, "*", receivedQuery.Get("fields"))
	})

	t.Run("returns error on http error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
		server := httptest.NewServer(handler)
		defer server.Close()

		restClient := &client.Client{
			HttpClient: server.Client(),
			BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
			AuthToken:  testTokenValue,
		}
		log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
		require.NoError(t, err)
		svc := New(restClient, log).(*Service)

		_, err = svc.GetAll(context.Background(), 0, "")
		assert.Error(t, err)
	})
}
