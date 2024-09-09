package event

import (
	"context"
	"go.uber.org/zap"

	"goflare.io/ember"
	"goflare.io/ignite"
	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/sqlc"
)

var _ Repository = (*repository)(nil)

type Repository interface {
	Create(ctx context.Context, customer *models.Event) error
	GetByID(ctx context.Context, id string) (*models.Event, error)
	MarkAsProcessed(ctx context.Context, id string) error
}

type repository struct {
	conn   driver.PostgresPool
	logger *zap.Logger
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger, cache *ember.MultiCache, poolManager ignite.Manager) (Repository, error) {
	return &repository{
		conn:   conn,
		logger: logger,
	}, nil
}

func (r *repository) Create(ctx context.Context, event *models.Event) error {
	return sqlc.New(r.conn).CreateEvent(ctx, sqlc.CreateEventParams{
		ID:        event.ID,
		Type:      event.Type,
		Processed: event.Processed,
	})
}

func (r *repository) GetByID(ctx context.Context, id string) (*models.Event, error) {
	sqlcEvent, err := sqlc.New(r.conn).GetEventByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &models.Event{
		ID:        sqlcEvent.ID,
		Type:      sqlcEvent.Type,
		Processed: sqlcEvent.Processed,
	}, nil
}

func (r *repository) MarkAsProcessed(ctx context.Context, id string) error {
	return sqlc.New(r.conn).MarkEventAsProcessed(ctx, sqlc.MarkEventAsProcessedParams{
		ID: id,
	})
}
