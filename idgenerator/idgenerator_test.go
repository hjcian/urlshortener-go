package idgenerator

import (
	"context"
	"goshorturl/pkg/concurrentstack"
	"goshorturl/repository"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type dbRecorder struct {
	repository.UnimplementedRepository
	wg          *sync.WaitGroup
	mu          sync.Mutex
	createCount int
	updateCount int
	selectCount int
}

func (d *dbRecorder) Create(ctx context.Context, id, url string, expiredAt time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.createCount++
	return nil
}

func (d *dbRecorder) Update(ctx context.Context, id, url string, expiredAt time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.updateCount++
	return nil
}

func (d *dbRecorder) SelectDeletedAndExpired(ctx context.Context, limit int) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	defer d.wg.Done()
	d.selectCount++
	return nil, nil
}

func TestIDGenerator_Get(t *testing.T) {
	t.Run("id stack is empty", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		db := &dbRecorder{wg: &wg}
		idgenerator := New(db, zap.NewNop())

		id, err := idgenerator.Get(context.Background(), "http://example.com", time.Now())
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Equal(t, 1, db.createCount)
		assert.Equal(t, 0, db.updateCount)

		// to wait the SelectDeletedAndExpired() to be called
		wg.Wait()
		assert.Equal(t, 1, db.selectCount)
	})
	t.Run("id stack has element", func(t *testing.T) {
		expected := "qwerty"
		stack := concurrentstack.New()
		stack.Push(expected)

		db := &dbRecorder{}
		idgenerator := &idGenerator{
			db:     db,
			logger: zap.NewNop(),
			ids:    stack,
		}

		id, err := idgenerator.Get(context.Background(), "http://example.com", time.Now())
		assert.NoError(t, err)
		assert.Equal(t, expected, id)
		assert.Equal(t, 0, db.createCount)
		assert.Equal(t, 1, db.updateCount)

		// no need to wait, because the SelectDeletedAndExpired() will not to be called
		assert.Equal(t, 0, db.selectCount)
	})
}

func TestValidate(t *testing.T) {
	idgenerator := New(&repository.UnimplementedRepository{}, zap.NewNop())
	generatedID, err := idgenerator.Get(context.Background(), "http://example.com", time.Now().Add(time.Hour))
	assert.NoError(t, err)

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			"Valid id",
			"aaaaaa",
			false,
		},
		{
			"Valid id from generator",
			generatedID,
			false,
		},
		{
			"empty id",
			"",
			true,
		},
		{
			"id too short",
			strings.Repeat("a", totalLetters-1),
			true,
		},
		{
			"id too long",
			strings.Repeat("a", totalLetters+1),
			true,
		},
		{
			"id contains invalid chars (!)",
			"!" + strings.Repeat("a", totalLetters-1),
			true,
		},
		{
			"id contains invalid chars (%)",
			"%" + strings.Repeat("a", totalLetters-1),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.id); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
