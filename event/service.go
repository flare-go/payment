package event

import (
	"context"
	"goflare.io/payment/models"
)

type Service interface {
	Create(ctx context.Context, event *models.Event) error
	IsEventProcessed(ctx context.Context, eventID string) (bool, error)
	MarkEventAsProcessed(ctx context.Context, eventID string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, event *models.Event) error {
	return s.repo.Create(ctx, event)
}

func (s *service) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return false, err
	}
	return event.Processed, nil
}

func (s *service) MarkEventAsProcessed(ctx context.Context, eventID string) error {
	return s.repo.MarkAsProcessed(ctx, eventID)
}
