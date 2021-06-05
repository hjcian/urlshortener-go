package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthController_Status(t *testing.T) {
	h := HealthController{}

	tests := []struct {
		name               string
		expectedStatusCode int
		expectedJSON       gin.H
	}{
		{
			"status ok",
			http.StatusOK,
			gin.H{"status": "ok"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(r)
			h.Status(ctx)

			assert.Equal(t, tt.expectedStatusCode, r.Code)
			var got gin.H
			err := json.Unmarshal(r.Body.Bytes(), &got)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedJSON, got)
		})
	}
}
