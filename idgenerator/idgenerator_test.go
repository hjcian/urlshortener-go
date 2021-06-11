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

// UPDATE "urls" SET "deleted_at"=NULL,"expired_at"='2021-08-09 09:20:41',"url"='https://example123.com',"updated_at"='2021-06-10 10:33:37.669' WHERE id = 'A4fNMF'
// SELECT "id" FROM "urls" WHERE deleted_at IS NOT NULL OR expired_at < '2021-06-10 10:27:54.923'

type recorder struct {
	repository.Repository
	mu          sync.Mutex
	createCount int
	updateCount int
	selectCount int
}

func (r *recorder) Create(ctx context.Context, id, url string, expiredAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCount++
	return nil
}

func (r *recorder) Update(ctx context.Context, id, url string, expiredAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updateCount++
	return nil
}

func (r *recorder) SelectDeletedAndExpired(ctx context.Context, limit int) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.selectCount++
	return nil, nil
}

func TestIDGenerator_Get(t *testing.T) {
	t.Run("id stack is empty", func(t *testing.T) {
		db := &recorder{}
		idgenerator := New(db, zap.NewNop())

		id, err := idgenerator.Get(context.Background(), "http://example.com", time.Now())
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Equal(t, 1, db.createCount)
		assert.Equal(t, 0, db.updateCount)

		time.Sleep(time.Second)
		assert.Equal(t, 1, db.selectCount)
	})
	t.Run("id stack has element", func(t *testing.T) {
		expected := "qwerty"
		stack := concurrentstack.New()
		stack.Push(expected)

		db := &recorder{}
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

		time.Sleep(time.Second)
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
