package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"goflare.io/payment/driver"
	"goflare.io/payment/models"
	"goflare.io/payment/models/enum"
)

type Service interface {
	Create(ctx context.Context, subscription *models.Subscription) error
	GetByID(ctx context.Context, id string) (*models.Subscription, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, subscription *models.Subscription) error
	Cancel(ctx context.Context, id string, cancelAtPeriodEnd bool) error
	List(ctx context.Context, customerID string, limit, offset uint64) ([]*models.Subscription, error)
	Renew(ctx context.Context, id string) error
	HandleExpiringSubscriptions(ctx context.Context) error
	Upsert(ctx context.Context, subscription *models.PartialSubscription) error
}

type service struct {
	repo               Repository
	transactionManager *driver.TransactionManager
	logger             *zap.Logger
}

func NewService(repo Repository, tm *driver.TransactionManager, logger *zap.Logger) Service {
	return &service{
		repo:               repo,
		transactionManager: tm,
		logger:             logger,
	}
}

func (s *service) Create(ctx context.Context, subscription *models.Subscription) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 在這裡可以添加額外的業務邏輯，例如檢查客戶是否有資格訂閱
		return s.repo.Create(ctx, tx, subscription)
	})
}

func (s *service) GetByID(ctx context.Context, id string) (*models.Subscription, error) {
	var subscription *models.Subscription
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		subscription, err = s.repo.GetByID(ctx, tx, id)
		return err
	})
	return subscription, err
}

func (s *service) Update(ctx context.Context, subscription *models.Subscription) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		existingSubscription, err := s.repo.GetByID(ctx, tx, subscription.ID)
		if err != nil {
			return fmt.Errorf("failed to get existing subscription: %w", err)
		}

		// 只更新允許修改的字段
		existingSubscription.ID = subscription.ID
		existingSubscription.PriceID = subscription.PriceID
		existingSubscription.Status = subscription.Status
		existingSubscription.CurrentPeriodStart = subscription.CurrentPeriodStart
		existingSubscription.CurrentPeriodEnd = subscription.CurrentPeriodEnd
		existingSubscription.CancelAtPeriodEnd = subscription.CancelAtPeriodEnd
		existingSubscription.TrialStart = subscription.TrialStart
		existingSubscription.TrialEnd = subscription.TrialEnd

		return s.repo.Update(ctx, tx, existingSubscription)
	})
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Delete(ctx, tx, id)
	})
}

func (s *service) Cancel(ctx context.Context, id string, cancelAtPeriodEnd bool) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		subscription, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get subscription: %w", err)
		}

		if subscription.Status == enum.SubscriptionStatusCanceled {
			return errors.New("subscription is already canceled")
		}

		return s.repo.Cancel(ctx, tx, id, cancelAtPeriodEnd)
	})
}

func (s *service) List(ctx context.Context, customerID string, limit, offset uint64) ([]*models.Subscription, error) {
	var subscriptions []*models.Subscription
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		subscriptions, err = s.repo.List(ctx, tx, customerID, limit, offset)
		return err
	})
	return subscriptions, err
}

func (s *service) Renew(ctx context.Context, id string) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		subscription, err := s.repo.GetByID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("failed to get subscription: %w", err)
		}

		if subscription.Status != enum.SubscriptionStatusActive {
			return errors.New("only active subscriptions can be renewed")
		}

		// 更新訂閱期間
		now := time.Now()
		subscription.CurrentPeriodStart = now
		subscription.CurrentPeriodEnd = now.AddDate(0, 1, 0) // 假設是月度訂閱，可以根據實際情況調整

		// 如果訂閱之前被設置為在期末取消，現在重新續訂後應該取消這個設置
		if subscription.CancelAtPeriodEnd {
			subscription.CancelAtPeriodEnd = false
		}

		// 如果有試用期，可能需要處理試用期相關邏輯
		if subscription.TrialEnd != nil && subscription.TrialEnd.After(now) {
			// 如果仍在試用期內，可能需要延長試用期或開始正式訂閱
			// 這裡的邏輯可以根據實際業務需求來調整
			subscription.TrialEnd = nil // 結束試用期
		}

		// 可能需要處理付款邏輯，例如通過 Stripe 創建新的 Invoice
		// 這裡只是一個示例，實際實現可能需要調用支付服務
		// err = s.paymentService.CreateInvoice(ctx, subscription)
		// if err != nil {
		//     return fmt.Errorf("failed to create invoice for renewed subscription: %w", err)
		// }

		// 更新訂閱狀態
		err = s.repo.Update(ctx, tx, subscription)
		if err != nil {
			return fmt.Errorf("failed to update subscription: %w", err)
		}

		// 可能需要觸發一些事件，例如發送郵件通知客戶訂閱已續期
		// s.eventEmitter.Emit("subscription.renewed", subscription)

		return nil
	})
}

func (s *service) HandleExpiringSubscriptions(ctx context.Context) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 獲取即將在下一天到期的訂閱
		// 注意：這裡需要在 Repository 中添加一個新的方法來獲取即將到期的訂閱
		expiringSubscriptions, err := s.repo.GetExpiringSubscriptions(ctx, tx, time.Now().AddDate(0, 0, 1))
		if err != nil {
			return fmt.Errorf("failed to get expiring subscriptions: %w", err)
		}

		for _, subscription := range expiringSubscriptions {
			if subscription.CancelAtPeriodEnd {
				// 如果訂閱設置為在期末取消，則取消訂閱
				err = s.Cancel(ctx, subscription.ID, true)
			} else {
				// 否則，嘗試續訂
				err = s.Renew(ctx, subscription.ID)
			}

			if err != nil {
				s.logger.Error("Failed to process expiring subscription",
					zap.Error(err),
					zap.String("subscriptionID", subscription.ID))
				// 可以選擇繼續處理其他訂閱，或者中斷整個過程
				// return err
			}
		}

		return nil
	})
}

func (s *service) Upsert(ctx context.Context, subscription *models.PartialSubscription) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.Upsert(ctx, tx, subscription)
	})
}
