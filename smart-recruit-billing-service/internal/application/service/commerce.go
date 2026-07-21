package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"smart-recruit-billing-service/internal/domain/model"
	"smart-recruit-billing-service/internal/infrastructure/payment"
	platformobservability "smart-recruit-platform-go/observability"
	"smart-recruit-proto/recruitment/pb"
)

type AlipayGateway interface {
	Environment() string
	PayURL(payment.PayRequest, time.Time) (string, error)
	VerifyNotification(map[string]string) error
	VerifyReturn(map[string]string) error
	Refund(context.Context, string, string, string, uint64, time.Time) (string, error)
	QueryRefund(context.Context, string, string, time.Time) (payment.RefundQueryResult, error)
	Query(context.Context, string, time.Time) (payment.QueryResult, error)
	Close(context.Context, string, time.Time) error
}

const billingOrderPaymentWindow = 30 * time.Minute

var (
	ErrPendingBillingOrder          = errors.New("an unpaid billing order already exists")
	ErrBillingOrderExpired          = errors.New("billing order has expired")
	ErrBillingOrderNotPayable       = errors.New("billing order cannot be paid")
	ErrBillingPaymentCompleted      = errors.New("billing payment already completed")
	ErrBillingPaymentPending        = errors.New("billing payment is still pending")
	ErrSubscriptionRenewalScheduled = errors.New("subscription renewal is already scheduled")
	ErrInvalidAlipayNotification    = errors.New("invalid Alipay notification")
)

type alipayCloseResolution uint8

const (
	alipayTradeClosed alipayCloseResolution = iota
	alipayTradePaid
)

// resolveAlipayCloseFailure handles the sandbox's inconsistent close surface.
// A missing trade means the generated cashier URL never created a channel
// trade, so the local attempt is safe to close and replace.
func resolveAlipayCloseFailure(closeErr error, result payment.QueryResult, queryErr error) (alipayCloseResolution, error) {
	if payment.IsTradeNotExist(queryErr) {
		return alipayTradeClosed, nil
	}
	if queryErr != nil {
		return alipayTradeClosed, fmt.Errorf("close Alipay trade: %v; verify status: %w", closeErr, queryErr)
	}
	switch result.TradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		return alipayTradePaid, nil
	case "TRADE_CLOSED":
		return alipayTradeClosed, nil
	default:
		return alipayTradeClosed, fmt.Errorf("close Alipay trade: %w (status %s)", closeErr, result.TradeStatus)
	}
}

func (s *Commerce) ReconcilePendingPayments(ctx context.Context, limit int) error {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	type attemptRow struct {
		PaymentID, OrderID         uint64
		MerchantOrderNo, PaymentNo string
		Status                     string
		AmountFen                  uint64
		ExpiresAt                  time.Time
		ReconcileAttempts          uint32
	}
	var attempts []attemptRow
	now := s.now().UTC()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`SELECT payment.id payment_id, payment.order_id, payment.merchant_order_no, payment.payment_no, payment.status, payment.amount_fen,
			COALESCE(payment.expires_at, orders.expires_at) expires_at, payment.reconcile_attempts
			FROM billing_payments payment JOIN billing_orders orders ON orders.id = payment.order_id
			WHERE payment.status IN ('pending','closing','unknown') AND payment.payment_environment = ?
			  AND (payment.next_reconcile_at IS NULL OR payment.next_reconcile_at <= ?)
			ORDER BY COALESCE(payment.next_reconcile_at, payment.created_at), payment.id LIMIT ? FOR UPDATE SKIP LOCKED`, s.Environment(), now, limit).Scan(&attempts).Error; err != nil {
			return err
		}
		if len(attempts) == 0 {
			return nil
		}
		ids := make([]uint64, 0, len(attempts))
		for _, attempt := range attempts {
			ids = append(ids, attempt.PaymentID)
		}
		return tx.Table("billing_payments").Where("id IN ?", ids).Updates(map[string]any{"next_reconcile_at": now.Add(10 * time.Minute), "updated_at": now}).Error
	}); err != nil {
		return err
	}
	for _, attempt := range attempts {
		result, err := s.alipay.Query(ctx, attempt.MerchantOrderNo, s.now())
		if err != nil {
			if payment.IsTradeNotExist(err) && (attempt.Status == "closing" || !attempt.ExpiresAt.After(now)) {
				if closeErr := s.closeReconciledPayment(ctx, attempt.PaymentID, now); closeErr != nil {
					return closeErr
				}
				continue
			}
			s.recordPaymentReconcileFailure(ctx, attempt.PaymentID, attempt.ReconcileAttempts, err)
			continue
		}
		if result.TradeStatus == "TRADE_SUCCESS" || result.TradeStatus == "TRADE_FINISHED" {
			if result.AmountFen != attempt.AmountFen {
				s.recordPaymentReconcileFailure(ctx, attempt.PaymentID, attempt.ReconcileAttempts, errors.New("Alipay payment amount mismatch"))
				continue
			}
			payload, _ := json.Marshal(result)
			digest := sha256.Sum256(payload)
			eventKey := "query:" + result.TradeNo + ":" + result.TradeStatus
			if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				if err := tx.Table("billing_webhook_events").Clauses(noOpConflict("event_key", "channel", "payment_environment", "event_key")).Create(map[string]any{"channel": "alipay", "payment_environment": s.Environment(), "event_key": eventKey, "event_type": "active_query", "signature_verified": true, "payload_sha256": hex.EncodeToString(digest[:]), "payload": string(payload), "status": "processed", "processed_at": now, "created_at": now}).Error; err != nil {
					return err
				}
				return s.settlePaymentTx(ctx, tx, attempt.PaymentID, result.TradeNo, result.AmountFen, now)
			}); err != nil {
				return fmt.Errorf("reconcile Alipay payment %s: %w", attempt.MerchantOrderNo, err)
			}
			continue
		}
		if result.TradeStatus == "TRADE_CLOSED" {
			if err := s.closeReconciledPayment(ctx, attempt.PaymentID, now); err != nil {
				return err
			}
			continue
		}
		if attempt.Status == "closing" || !attempt.ExpiresAt.After(now) {
			if err := s.alipay.Close(ctx, attempt.MerchantOrderNo, s.now()); err != nil {
				s.recordPaymentReconcileFailure(ctx, attempt.PaymentID, attempt.ReconcileAttempts, err)
				continue
			}
			if err := s.closeReconciledPayment(ctx, attempt.PaymentID, now); err != nil {
				return err
			}
		} else {
			_ = s.db.WithContext(ctx).Table("billing_payments").Where("id = ?", attempt.PaymentID).Updates(map[string]any{"last_queried_at": now, "last_reconcile_error": nil, "next_reconcile_at": now.Add(2 * time.Minute), "updated_at": now}).Error
		}
	}
	if err := s.closeExpiredOrders(ctx, nil); err != nil {
		return err
	}
	return s.ReconcilePendingRefunds(ctx, limit)
}

func (s *Commerce) recordPaymentReconcileFailure(ctx context.Context, paymentID uint64, attempts uint32, cause error) {
	delay := time.Minute * time.Duration(1<<minInt(int(attempts), 6))
	now := s.now().UTC()
	_ = s.db.WithContext(ctx).Table("billing_payments").Where("id = ?", paymentID).Updates(map[string]any{"status": "unknown", "reconcile_attempts": gorm.Expr("reconcile_attempts + 1"), "last_reconcile_error": truncateError(cause), "last_queried_at": now, "next_reconcile_at": now.Add(delay), "updated_at": now}).Error
}

func (s *Commerce) closeReconciledPayment(ctx context.Context, paymentID uint64, now time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var orderID uint64
		if err := tx.Table("billing_payments").Where("id = ?", paymentID).Select("order_id").Scan(&orderID).Error; err != nil {
			return err
		}
		if err := tx.Table("billing_payments").Where("id = ? AND status IN ('created','pending','closing','unknown')", paymentID).Updates(map[string]any{"status": "closed", "active_slot": nil, "closed_at": now, "next_reconcile_at": nil, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Table("billing_orders").Where("id = ? AND status IN ('pending','paying')", orderID).Updates(map[string]any{"status": "closed", "active_slot": nil, "closed_at": now, "updated_at": now}).Error
	})
}

func (s *Commerce) ReconcilePendingRefunds(ctx context.Context, limit int) error {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	type refundRow struct {
		RefundNo, MerchantOrderNo, Reason string
		AmountFen                         uint64
		Attempts                          uint32 `gorm:"column:reconcile_attempts"`
	}
	var refunds []refundRow
	now := s.now().UTC()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`SELECT refund.refund_no, payment.merchant_order_no, refund.reason, refund.amount_fen, refund.reconcile_attempts
			FROM billing_refunds refund JOIN billing_payments payment ON payment.id = refund.payment_id
			WHERE refund.status IN ('processing','unknown') AND refund.next_reconcile_at <= ?
			ORDER BY refund.next_reconcile_at, refund.id LIMIT ? FOR UPDATE SKIP LOCKED`, now, limit).Scan(&refunds).Error; err != nil {
			return err
		}
		if len(refunds) == 0 {
			return nil
		}
		numbers := make([]string, 0, len(refunds))
		for _, refund := range refunds {
			numbers = append(numbers, refund.RefundNo)
		}
		return tx.Table("billing_refunds").Where("refund_no IN ?", numbers).Updates(map[string]any{"next_reconcile_at": now.Add(10 * time.Minute), "updated_at": now}).Error
	}); err != nil {
		return err
	}
	for _, refund := range refunds {
		result, err := s.alipay.QueryRefund(ctx, refund.MerchantOrderNo, refund.RefundNo, s.now())
		if err != nil {
			if payment.IsTradeNotExist(err) && refund.Attempts >= 2 {
				channelNo, retryErr := s.alipay.Refund(ctx, refund.MerchantOrderNo, refund.RefundNo, refund.Reason, refund.AmountFen, s.now())
				if retryErr == nil {
					if err := s.finalizeRefund(ctx, refund.RefundNo, channelNo, now); err != nil {
						return err
					}
					continue
				}
				if payment.IsAPIRejected(retryErr) {
					if err := s.failRefund(ctx, refund.RefundNo, retryErr, now); err != nil {
						return err
					}
					continue
				}
				err = retryErr
			}
			delay := time.Minute * time.Duration(1<<minInt(int(refund.Attempts), 6))
			_ = s.db.WithContext(ctx).Table("billing_refunds").Where("refund_no = ?", refund.RefundNo).Updates(map[string]any{"status": "unknown", "reconcile_attempts": gorm.Expr("reconcile_attempts + 1"), "last_error": truncateError(err), "next_reconcile_at": now.Add(delay), "updated_at": now}).Error
			continue
		}
		if result.RefundNo != refund.RefundNo || result.AmountFen != refund.AmountFen {
			_ = s.db.WithContext(ctx).Table("billing_refunds").Where("refund_no = ?", refund.RefundNo).Updates(map[string]any{"status": "unknown", "reconcile_attempts": gorm.Expr("reconcile_attempts + 1"), "last_error": "Alipay refund query identity or amount mismatch", "next_reconcile_at": now.Add(15 * time.Minute), "updated_at": now}).Error
			continue
		}
		if result.RefundStatus == "REFUND_SUCCESS" {
			if err := s.finalizeRefund(ctx, refund.RefundNo, result.TradeNo, now); err != nil {
				return err
			}
			continue
		}
		_ = s.db.WithContext(ctx).Table("billing_refunds").Where("refund_no = ?", refund.RefundNo).Updates(map[string]any{"last_error": nil, "next_reconcile_at": now.Add(5 * time.Minute), "updated_at": now}).Error
	}
	return nil
}

func (s *Commerce) failRefund(ctx context.Context, refundNo string, cause error, now time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var orderID uint64
		if err := tx.Table("billing_refunds").Where("refund_no = ?", refundNo).Select("order_id").Scan(&orderID).Error; err != nil {
			return err
		}
		if err := tx.Table("billing_refunds").Where("refund_no = ? AND status IN ('processing','unknown')", refundNo).Updates(map[string]any{"status": "failed", "active_slot": nil, "next_reconcile_at": nil, "last_error": truncateError(cause), "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Table("billing_orders").Where("id = ? AND status = 'refunding'", orderID).Updates(map[string]any{"status": "paid", "updated_at": now}).Error
	})
}

func (s *Commerce) finalizeRefund(ctx context.Context, refundNo, channelNo string, now time.Time) error {
	type row struct {
		OrderID, PaymentID, OwnerID uint64
		OwnerType, Reason           string
	}
	var refund row
	if err := s.db.WithContext(ctx).Raw(`SELECT refund.order_id, refund.payment_id, refund.reason, orders.owner_type, orders.owner_id
		FROM billing_refunds refund JOIN billing_orders orders ON orders.id = refund.order_id WHERE refund.refund_no = ?`, refundNo).Scan(&refund).Error; err != nil {
		return err
	}
	if refund.OrderID == 0 {
		return gorm.ErrRecordNotFound
	}
	owner := model.Owner{Type: model.OwnerType(refund.OwnerType), ID: refund.OwnerID}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var statusValue string
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("billing_refunds").Where("refund_no = ?", refundNo).Select("status").Scan(&statusValue).Error; err != nil {
			return err
		}
		if statusValue == "succeeded" {
			return nil
		}
		if err := tx.Table("billing_refunds").Where("refund_no = ?", refundNo).Updates(map[string]any{"status": "succeeded", "active_slot": nil, "channel_refund_no": channelNo, "succeeded_at": now, "next_reconcile_at": nil, "last_error": nil, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Table("billing_orders").Where("id = ?", refund.OrderID).Update("status", "refunded").Error; err != nil {
			return err
		}
		if err := tx.Table("billing_payments").Where("id = ?", refund.PaymentID).Update("status", "refunded").Error; err != nil {
			return err
		}
		var subscriptionIDs []uint64
		if err := tx.Table("billing_subscriptions").Where("activated_by_order_id = ?", refund.OrderID).Pluck("id", &subscriptionIDs).Error; err != nil {
			return err
		}
		var grants []struct{ ID, RemainingCredits uint64 }
		grantQuery := tx.Table("ai_credit_grants").Where("owner_type = ? AND owner_id = ? AND ((source_type = 'order' AND source_id = ?) OR (source_type = 'subscription' AND source_id IN ?)) AND status = 'active'", owner.Type, owner.ID, refund.OrderID, subscriptionIDs)
		if err := grantQuery.Find(&grants).Error; err != nil {
			return err
		}
		var balance uint64
		if err := tx.Table("ai_credit_grants").Where("owner_type = ? AND owner_id = ? AND status = 'active' AND valid_from <= ? AND (expires_at IS NULL OR expires_at > ?)", owner.Type, owner.ID, now, now).Select("COALESCE(SUM(remaining_credits), 0)").Scan(&balance).Error; err != nil {
			return err
		}
		for _, grant := range grants {
			if grant.RemainingCredits == 0 {
				continue
			}
			if grant.RemainingCredits <= balance {
				balance -= grant.RemainingCredits
			} else {
				balance = 0
			}
			if err := tx.Table("ai_credit_ledger").Create(map[string]any{"entry_id": uuid.NewString(), "owner_type": owner.Type, "owner_id": owner.ID, "grant_id": grant.ID, "entry_type": "refund", "credits_delta": -int64(grant.RemainingCredits), "balance_after": int64(balance), "idempotency_key": fmt.Sprintf("refund:%s:grant:%d", refundNo, grant.ID), "description": refund.Reason, "created_at": now}).Error; err != nil {
				return err
			}
		}
		if err := grantQuery.Updates(map[string]any{"status": "revoked", "remaining_credits": 0, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Table("billing_subscriptions").Where("activated_by_order_id = ? AND status = 'active'", refund.OrderID).Updates(map[string]any{"status": "cancelled", "updated_at": now}).Error
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Commerce struct {
	db     *gorm.DB
	alipay AlipayGateway
	now    func() time.Time
}

type commerceOrderRow struct {
	ID             uint64 `gorm:"column:id"`
	OwnerID        uint64 `gorm:"column:owner_id"`
	ProductID      uint64 `gorm:"column:product_id"`
	PriceVersionID uint64 `gorm:"column:price_version_id"`
	OwnerType      string `gorm:"column:owner_type"`
	OrderNo        string `gorm:"column:order_no"`
	OrderType      string `gorm:"column:order_type"`
	Status         string `gorm:"column:status"`
	Environment    string `gorm:"column:payment_environment"`
	Currency       string `gorm:"column:currency"`
	AmountFen      uint64 `gorm:"column:amount_fen"`
}

func NewCommerce(db *gorm.DB, alipay AlipayGateway) (*Commerce, error) {
	if db == nil {
		return nil, errors.New("commerce database is required")
	}
	if alipay == nil {
		return nil, errors.New("Alipay sandbox gateway is required")
	}
	if alipay.Environment() != "sandbox" {
		return nil, errors.New("development commerce requires Alipay sandbox")
	}
	return &Commerce{db: db, alipay: alipay, now: time.Now}, nil
}

func (s *Commerce) Environment() string { return s.alipay.Environment() }

func (s *Commerce) ListCatalog(ctx context.Context, owner model.Owner) ([]*pb.BillingProductInfo, error) {
	type row struct {
		ProductID                                  int64
		ProductKey, Name, Description, ProductType string
		PriceID                                    int64
		Version                                    int32
		BillingTerm                                string
		AmountFen, IncludedCredits                 int64
		Currency, Snapshot                         string
	}
	var rows []row
	err := s.db.WithContext(ctx).Raw(`SELECT product.id product_id, product.product_key, product.name, COALESCE(product.description, '') description, product.product_type,
		price.id price_id, price.version, price.billing_term, price.amount_fen, price.currency, price.included_credits, COALESCE(CAST(price.entitlement_snapshot AS CHAR), '') snapshot
		FROM billing_products product JOIN billing_price_versions price ON price.product_id = product.id
		WHERE product.owner_type = ? AND product.status = 'active' AND price.status = 'published'
		  AND price.effective_at <= UTC_TIMESTAMP(3) AND (price.retired_at IS NULL OR price.retired_at > UTC_TIMESTAMP(3))
		ORDER BY product.id, price.amount_fen, price.version DESC`, owner.Type).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	byID := map[int64]*pb.BillingProductInfo{}
	result := make([]*pb.BillingProductInfo, 0)
	for _, row := range rows {
		product := byID[row.ProductID]
		if product == nil {
			product = &pb.BillingProductInfo{Id: row.ProductID, ProductKey: row.ProductKey, Name: row.Name, Description: row.Description, ProductType: row.ProductType}
			byID[row.ProductID] = product
			result = append(result, product)
		}
		product.Prices = append(product.Prices, &pb.BillingPriceInfo{Id: row.PriceID, Version: row.Version, BillingTerm: row.BillingTerm, AmountFen: row.AmountFen, Currency: row.Currency, IncludedCredits: row.IncludedCredits, EntitlementSnapshotJson: row.Snapshot})
	}
	return result, nil
}

func (s *Commerce) ListAdminCatalog(ctx context.Context) ([]*pb.BillingProductInfo, error) {
	type row struct {
		ProductID                                  int64
		ProductKey, Name, Description, ProductType string
		PriceID                                    *int64
		Version                                    *int32
		BillingTerm                                *string
		AmountFen, IncludedCredits                 *int64
		Currency, Snapshot                         *string
	}
	var rows []row
	err := s.db.WithContext(ctx).Raw(`SELECT product.id product_id, product.product_key, product.name, COALESCE(product.description, '') description, product.product_type,
		price.id price_id, price.version, price.billing_term, price.amount_fen, price.currency, price.included_credits, CAST(price.entitlement_snapshot AS CHAR) snapshot
		FROM billing_products product LEFT JOIN billing_price_versions price ON price.product_id = product.id
		ORDER BY product.owner_type, product.id, price.version DESC`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	byID := map[int64]*pb.BillingProductInfo{}
	result := make([]*pb.BillingProductInfo, 0)
	for _, row := range rows {
		product := byID[row.ProductID]
		if product == nil {
			product = &pb.BillingProductInfo{Id: row.ProductID, ProductKey: row.ProductKey, Name: row.Name, Description: row.Description, ProductType: row.ProductType}
			byID[row.ProductID] = product
			result = append(result, product)
		}
		if row.PriceID != nil {
			product.Prices = append(product.Prices, &pb.BillingPriceInfo{Id: *row.PriceID, Version: *row.Version, BillingTerm: ptrValue(row.BillingTerm), AmountFen: ptrInt64(row.AmountFen), Currency: ptrValue(row.Currency), IncludedCredits: ptrInt64(row.IncludedCredits), EntitlementSnapshotJson: ptrValue(row.Snapshot)})
		}
	}
	return result, nil
}

func (s *Commerce) SavePriceVersion(ctx context.Context, productID, priceVersionID uint64, term string, amountFen, credits uint64, snapshot string, publish bool) (*pb.BillingPriceInfo, error) {
	if productID == 0 || amountFen == 0 || credits == 0 || (term != "monthly" && term != "yearly" && term != "one_time") {
		return nil, errors.New("product, positive amount, credits and billing term are required")
	}
	if strings.TrimSpace(snapshot) == "" {
		snapshot = `{}`
	}
	if !json.Valid([]byte(snapshot)) {
		return nil, errors.New("entitlement snapshot must be valid JSON")
	}
	var result pb.BillingPriceInfo
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product struct {
			ProductKey  string
			OwnerType   string
			ProductType string
		}
		if err := tx.Table("billing_products").Where("id = ?", productID).Select("product_key, owner_type, product_type").Scan(&product).Error; err != nil {
			return err
		}
		if product.ProductType == "" {
			return gorm.ErrRecordNotFound
		}
		if product.ProductType == "credit_pack" && term != "one_time" {
			return errors.New("credit packs must use one_time billing")
		}
		if product.ProductType == "subscription" && term == "one_time" {
			return errors.New("subscriptions must use monthly or yearly billing")
		}
		if product.ProductType == "subscription" {
			generated, err := buildPublishedSubscriptionSnapshot(tx, product.ProductKey, product.OwnerType, credits)
			if err != nil {
				return err
			}
			snapshot = generated
		} else {
			snapshot = `{}`
		}
		statusValue := "draft"
		var effectiveAt any = nil
		if publish {
			statusValue = "published"
			// DATETIME values are compared with UTC_TIMESTAMP throughout Billing.
			// Use the database UTC clock so loc=Local cannot shift an immediate
			// publication eight hours into the future.
			effectiveAt = gorm.Expr("UTC_TIMESTAMP(3)")
		}
		if priceVersionID > 0 {
			var existing struct {
				Status  string
				Version int32
			}
			if err := tx.Table("billing_price_versions").Where("id = ? AND product_id = ?", priceVersionID, productID).Scan(&existing).Error; err != nil {
				return err
			}
			if existing.Status != "draft" {
				return errors.New("published price versions are immutable")
			}
			if err := tx.Table("billing_price_versions").Where("id = ?", priceVersionID).Updates(map[string]any{"billing_term": term, "amount_fen": amountFen, "included_credits": credits, "entitlement_snapshot": snapshot, "status": statusValue, "effective_at": effectiveAt, "updated_at": gorm.Expr("UTC_TIMESTAMP(3)")}).Error; err != nil {
				return err
			}
			result.Id = int64(priceVersionID)
			result.Version = existing.Version
		} else {
			var version int32
			if err := tx.Table("billing_price_versions").Where("product_id = ?", productID).Select("COALESCE(MAX(version), 0) + 1").Scan(&version).Error; err != nil {
				return err
			}
			row := map[string]any{"product_id": productID, "version": version, "billing_term": term, "amount_fen": amountFen, "currency": "CNY", "included_credits": credits, "entitlement_snapshot": snapshot, "status": statusValue, "effective_at": effectiveAt, "created_at": gorm.Expr("UTC_TIMESTAMP(3)"), "updated_at": gorm.Expr("UTC_TIMESTAMP(3)")}
			if err := tx.Table("billing_price_versions").Create(row).Error; err != nil {
				return err
			}
			var id int64
			if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&id).Error; err != nil {
				return err
			}
			result.Id = id
			result.Version = version
		}
		if publish {
			if err := tx.Table("billing_price_versions").Where("product_id = ? AND billing_term = ? AND id <> ? AND status = 'published'", productID, term, result.Id).Updates(map[string]any{"status": "retired", "retired_at": gorm.Expr("UTC_TIMESTAMP(3)"), "updated_at": gorm.Expr("UTC_TIMESTAMP(3)")}).Error; err != nil {
				return err
			}
			if err := tx.Table("billing_products").Where("id = ?", productID).Updates(map[string]any{"status": "active", "updated_at": gorm.Expr("UTC_TIMESTAMP(3)")}).Error; err != nil {
				return err
			}
		}
		result.BillingTerm = term
		result.AmountFen = int64(amountFen)
		result.Currency = "CNY"
		result.IncludedCredits = int64(credits)
		result.EntitlementSnapshotJson = snapshot
		return nil
	})
	return &result, err
}

// buildPublishedSubscriptionSnapshot makes the server the only authority that
// binds a sellable subscription price to immutable AI capability releases.
// The admin UI therefore cannot accidentally publish stale or arbitrary JSON.
func buildPublishedSubscriptionSnapshot(tx *gorm.DB, productKey, ownerType string, credits uint64) (string, error) {
	snapshot := map[string]json.RawMessage{}
	if ownerType == "tenant" {
		planKey := strings.TrimPrefix(productKey, "tenant_")
		type entitlementRow struct {
			EntitlementKey string
			ValueJSON      string
		}
		var rows []entitlementRow
		err := tx.Raw(`
			SELECT entitlement.entitlement_key, CAST(entitlement.value_json AS CHAR) value_json
			FROM platform_plan_versions version
			JOIN platform_plans plan ON plan.id = version.plan_id
			JOIN platform_plan_entitlements entitlement ON entitlement.plan_version_id = version.id
			WHERE plan.plan_key = ? AND version.status = 'published'
			  AND version.version = (
			    SELECT MAX(latest.version)
			    FROM platform_plan_versions latest
			    WHERE latest.plan_id = version.plan_id AND latest.status = 'published'
			  )`, planKey).Scan(&rows).Error
		if err != nil {
			return "", err
		}
		if len(rows) == 0 {
			return "", fmt.Errorf("no published platform plan found for product %s", productKey)
		}
		for _, row := range rows {
			if !json.Valid([]byte(row.ValueJSON)) {
				return "", fmt.Errorf("invalid entitlement %s in published platform plan", row.EntitlementKey)
			}
			snapshot[row.EntitlementKey] = json.RawMessage(row.ValueJSON)
		}
	} else if ownerType == "user" {
		var releaseID uint64
		err := tx.Raw(`
			SELECT current_published_version_id
			FROM platform_ai_capabilities
			WHERE capability_key = 'ai.chat' AND audience = 'candidate' AND status = 'active'
			LIMIT 1`).Scan(&releaseID).Error
		if err != nil {
			return "", err
		}
		if releaseID == 0 {
			return "", errors.New("candidate AI capability has no published release")
		}
		snapshot["ai.chat.enabled"] = json.RawMessage("true")
		snapshot["ai.chat.release_version_id"] = json.RawMessage(strconv.FormatUint(releaseID, 10))
	} else {
		return "", fmt.Errorf("unsupported subscription owner type %s", ownerType)
	}
	snapshot["ai.credits.monthly"] = json.RawMessage(strconv.FormatUint(credits, 10))
	encoded, err := json.Marshal(snapshot)
	return string(encoded), err
}

func (s *Commerce) ListRateCards(ctx context.Context) ([]*pb.AIRateCardInfo, error) {
	type row struct {
		ID                                      int64
		ProviderKey, ModelKey, Currency, Status string
		Version                                 int32
		Input, Output, Cached, Credit           int64
		EffectiveAt                             *time.Time
	}
	var rows []row
	err := s.db.WithContext(ctx).Raw(`SELECT id, provider_key, model_key, version, currency, status,
		input_micros_per_1k_tokens input, output_micros_per_1k_tokens output,
		cached_input_micros_per_1k_tokens cached, credit_micros credit, effective_at
		FROM ai_rate_cards ORDER BY provider_key, model_key, version DESC`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]*pb.AIRateCardInfo, 0, len(rows))
	for _, row := range rows {
		item := &pb.AIRateCardInfo{Id: row.ID, ProviderKey: row.ProviderKey, ModelKey: row.ModelKey, Version: row.Version, Currency: row.Currency, InputMicrosPer_1KTokens: row.Input, OutputMicrosPer_1KTokens: row.Output, CachedInputMicrosPer_1KTokens: row.Cached, CreditMicros: row.Credit, Status: row.Status}
		if row.EffectiveAt != nil {
			item.EffectiveAtUnixMs = row.EffectiveAt.UnixMilli()
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *Commerce) SaveRateCard(ctx context.Context, providerKey, modelKey string, input, output, cached, credit uint64, publish bool) (*pb.AIRateCardInfo, error) {
	providerKey = strings.TrimSpace(providerKey)
	modelKey = strings.TrimSpace(modelKey)
	if providerKey == "" || modelKey == "" || credit == 0 || (input == 0 && output == 0) {
		return nil, errors.New("provider, model, supplier rates and positive credit conversion are required")
	}
	now := s.now().UTC()
	statusValue := "draft"
	var effectiveAt any = nil
	if publish {
		statusValue = "published"
		effectiveAt = gorm.Expr("UTC_TIMESTAMP(3)")
	}
	var result pb.AIRateCardInfo
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target struct {
			ProviderKey string
			ModelKey    string
		}
		if err := tx.Table("llm_models model").
			Joins("JOIN llm_providers provider ON provider.id = model.provider_id").
			Where("provider.name = ? AND model.model_name = ? AND provider.is_enabled = ? AND model.is_enabled = ?", providerKey, modelKey, true, true).
			Select("provider.name provider_key, model.model_name model_key").Take(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("provider and model must match an enabled platform LLM configuration")
			}
			return fmt.Errorf("validate maintained AI model: %w", err)
		}
		// Persist the catalog's canonical spelling even on case-insensitive MySQL
		// collations so Billing keys remain byte-for-byte equal to runtime audit keys.
		providerKey = target.ProviderKey
		modelKey = target.ModelKey
		var version int32
		if err := tx.Table("ai_rate_cards").Where("provider_key = ? AND model_key = ?", providerKey, modelKey).Select("COALESCE(MAX(version), 0) + 1").Scan(&version).Error; err != nil {
			return err
		}
		row := map[string]any{"provider_key": providerKey, "model_key": modelKey, "version": version, "currency": "CNY", "input_micros_per_1k_tokens": input, "output_micros_per_1k_tokens": output, "cached_input_micros_per_1k_tokens": cached, "credit_micros": credit, "status": statusValue, "effective_at": effectiveAt, "created_at": gorm.Expr("UTC_TIMESTAMP(3)")}
		if err := tx.Table("ai_rate_cards").Create(row).Error; err != nil {
			return err
		}
		var id int64
		if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&id).Error; err != nil {
			return err
		}
		if publish {
			if err := tx.Table("ai_rate_cards").Where("provider_key = ? AND model_key = ? AND id <> ? AND status = 'published'", providerKey, modelKey, id).Updates(map[string]any{"status": "retired", "retired_at": gorm.Expr("UTC_TIMESTAMP(3)")}).Error; err != nil {
				return err
			}
		}
		result = pb.AIRateCardInfo{Id: id, ProviderKey: providerKey, ModelKey: modelKey, Version: version, Currency: "CNY", InputMicrosPer_1KTokens: int64(input), OutputMicrosPer_1KTokens: int64(output), CachedInputMicrosPer_1KTokens: int64(cached), CreditMicros: int64(credit), Status: statusValue}
		if publish {
			result.EffectiveAtUnixMs = now.UnixMilli()
		}
		return nil
	})
	return &result, err
}

func (s *Commerce) CurrentSubscription(ctx context.Context, owner model.Owner) (*pb.BillingSubscriptionInfo, error) {
	type row struct {
		ID                                    int64
		ProductKey, ProductName, Status, Term string
		IncludedCredits                       int64
		Start, End                            time.Time
		Cancel                                bool
	}
	var value row
	err := s.db.WithContext(ctx).Raw(`SELECT subscription.id, product.product_key, product.name product_name, subscription.status, subscription.term,
		price.included_credits,
		subscription.current_period_start start, subscription.current_period_end end, subscription.cancel_at_period_end cancel
		FROM billing_subscriptions subscription
		JOIN billing_products product ON product.id = subscription.product_id
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = ? AND subscription.owner_id = ? AND subscription.status = 'active'
		ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, owner.Type, owner.ID).Scan(&value).Error
	if err != nil {
		return nil, err
	}
	if value.ID > 0 {
		return &pb.BillingSubscriptionInfo{
			Id: value.ID, ProductKey: value.ProductKey, ProductName: value.ProductName,
			Status: value.Status, Term: value.Term, CurrentPeriodStartUnixMs: value.Start.UnixMilli(),
			CurrentPeriodEndUnixMs: value.End.UnixMilli(), CancelAtPeriodEnd: value.Cancel,
			Source: "paid_subscription", IncludedCredits: value.IncludedCredits,
		}, nil
	}

	if owner.Type == model.OwnerTenant {
		type tenantPlanRow struct {
			ID                      int64
			ProductKey, ProductName string
			Status                  string
			IncludedCredits         int64
			Start                   time.Time
			End                     *time.Time
		}
		var plan tenantPlanRow
		err = s.db.WithContext(ctx).Raw(`SELECT subscription.id, plan.plan_key product_key, plan.name product_name,
			subscription.status, subscription.starts_at start, subscription.ends_at end,
			COALESCE(CAST(JSON_UNQUOTE(entitlement.value_json) AS SIGNED), 0) included_credits
			FROM tenant_subscriptions subscription
			JOIN platform_plan_versions version ON version.id = subscription.plan_version_id
			JOIN platform_plans plan ON plan.id = version.plan_id
			LEFT JOIN platform_plan_entitlements entitlement
			  ON entitlement.plan_version_id = version.id AND entitlement.entitlement_key = 'ai.credits.monthly'
			WHERE subscription.tenant_id = ? AND subscription.status = 'active'
			  AND subscription.starts_at <= UTC_TIMESTAMP(3)
			  AND (subscription.ends_at IS NULL OR subscription.ends_at > UTC_TIMESTAMP(3))
			ORDER BY subscription.starts_at DESC, subscription.id DESC LIMIT 1`, owner.ID).Scan(&plan).Error
		if err != nil {
			return nil, err
		}
		if plan.ID == 0 {
			return nil, nil
		}
		result := &pb.BillingSubscriptionInfo{
			Id: plan.ID, ProductKey: plan.ProductKey, ProductName: plan.ProductName,
			Status: plan.Status, Term: "monthly", CurrentPeriodStartUnixMs: plan.Start.UnixMilli(),
			Source: "platform_plan", IncludedCredits: plan.IncludedCredits,
		}
		if plan.End != nil {
			result.CurrentPeriodEndUnixMs = plan.End.UnixMilli()
		}
		return result, nil
	}

	type freeTierRow struct {
		ID                                    int64
		ProductKey, ProductName, Status, Term string
		IncludedCredits                       int64
	}
	var freeTier freeTierRow
	err = s.db.WithContext(ctx).Raw(`SELECT price.id, product.product_key, product.name product_name,
		'active' status, price.billing_term term, price.included_credits
		FROM billing_products product
		JOIN billing_price_versions price ON price.product_id = product.id
		WHERE product.product_key = 'candidate_free' AND product.owner_type = 'user'
		  AND product.status = 'active' AND price.status = 'published'
		  AND price.effective_at <= UTC_TIMESTAMP(3)
		  AND (price.retired_at IS NULL OR price.retired_at > UTC_TIMESTAMP(3))
		ORDER BY price.version DESC LIMIT 1`).Scan(&freeTier).Error
	if err != nil {
		return nil, err
	}
	if freeTier.ID == 0 {
		return nil, nil
	}
	start, end := shanghaiBillingMonth(s.now())
	return &pb.BillingSubscriptionInfo{
		Id: freeTier.ID, ProductKey: freeTier.ProductKey, ProductName: freeTier.ProductName,
		Status: freeTier.Status, Term: freeTier.Term, CurrentPeriodStartUnixMs: start.UnixMilli(),
		CurrentPeriodEndUnixMs: end.UnixMilli(), Source: "free_tier", IncludedCredits: freeTier.IncludedCredits,
	}, nil
}

// ScheduledSubscription returns the next paid subscription period that has
// already been purchased but has not reached its activation time yet.
func (s *Commerce) ScheduledSubscription(ctx context.Context, owner model.Owner) (*pb.BillingSubscriptionInfo, error) {
	type row struct {
		ID                                    int64
		ProductKey, ProductName, Status, Term string
		IncludedCredits                       int64
		Start, End                            time.Time
		Cancel                                bool
	}
	var value row
	err := s.db.WithContext(ctx).Raw(`SELECT subscription.id, product.product_key, product.name product_name, subscription.status, subscription.term,
		price.included_credits,
		subscription.current_period_start start, subscription.current_period_end end, subscription.cancel_at_period_end cancel
		FROM billing_subscriptions subscription
		JOIN billing_products product ON product.id = subscription.product_id
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = ? AND subscription.owner_id = ? AND subscription.status = 'pending'
		  AND subscription.current_period_end > ?
		ORDER BY subscription.current_period_start, subscription.id LIMIT 1`, owner.Type, owner.ID, s.now().UTC()).Scan(&value).Error
	if err != nil {
		return nil, err
	}
	if value.ID == 0 {
		return nil, nil
	}
	return &pb.BillingSubscriptionInfo{
		Id: value.ID, ProductKey: value.ProductKey, ProductName: value.ProductName,
		Status: value.Status, Term: value.Term, CurrentPeriodStartUnixMs: value.Start.UnixMilli(),
		CurrentPeriodEndUnixMs: value.End.UnixMilli(), CancelAtPeriodEnd: value.Cancel,
		Source: "paid_subscription", IncludedCredits: value.IncludedCredits,
	}, nil
}

// CurrentCreditSummary returns gross and consumed credits for current grant
// buckets, including exhausted buckets so a fully consumed package remains
// visible as 100% used. Reservations are excluded because they are unsettled.
const currentCreditSummarySQL = `SELECT
		COALESCE(SUM(total_credits), 0) total,
		COALESCE(SUM(remaining_credits), 0) remaining
		FROM ai_credit_grants
		WHERE owner_type = ? AND owner_id = ? AND status IN ('active', 'exhausted')
		  AND valid_from <= ?
		  AND (expires_at IS NULL OR expires_at > ?)`

func (s *Commerce) CurrentCreditSummary(ctx context.Context, owner model.Owner) (total, used int64, err error) {
	type row struct {
		Total     int64
		Remaining int64
	}
	var value row
	now := s.now().UTC()
	err = s.db.WithContext(ctx).Raw(currentCreditSummarySQL, owner.Type, owner.ID, now, now).Scan(&value).Error
	if err != nil {
		return 0, 0, err
	}
	used = value.Total - value.Remaining
	if used < 0 {
		used = 0
	}
	return value.Total, used, nil
}

func (s *Commerce) NextRefreshAt(subscription *pb.BillingSubscriptionInfo) int64 {
	if subscription == nil {
		return 0
	}
	_, end := shanghaiBillingMonth(s.now())
	if subscription.CurrentPeriodEndUnixMs > 0 && subscription.CurrentPeriodEndUnixMs < end.UnixMilli() {
		return subscription.CurrentPeriodEndUnixMs
	}
	return end.UnixMilli()
}

func shanghaiBillingMonth(now time.Time) (time.Time, time.Time) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
	return start.UTC(), start.AddDate(0, 1, 0).UTC()
}

// closeExpiredOrders synchronously projects the time-based order state before
// reads and payment attempts. The maintenance loop remains the safety net, but
// callers never have to wait for its next cycle to observe a closed order.
func (s *Commerce) closeExpiredOrders(ctx context.Context, owner *model.Owner) error {
	now := s.now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Table("billing_orders").Where("status IN ('pending','paying') AND expires_at <= ?", now)
		if owner != nil {
			query = query.Where("owner_type = ? AND owner_id = ?", owner.Type, owner.ID)
		}
		var orders []struct{ ID uint64 }
		if err := query.Select("id").Scan(&orders).Error; err != nil {
			return err
		}
		if len(orders) == 0 {
			return nil
		}
		for _, order := range orders {
			var activeCount int64
			if err := tx.Table("billing_payments").Where("order_id = ? AND active_slot = 1", order.ID).Count(&activeCount).Error; err != nil {
				return err
			}
			if activeCount > 0 {
				if err := tx.Table("billing_payments").Where("order_id = ? AND active_slot = 1 AND status IN ('created','pending','unknown')", order.ID).Updates(map[string]any{"status": "closing", "next_reconcile_at": now, "updated_at": now}).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Table("billing_orders").Where("id = ? AND status IN ('pending','paying')", order.ID).Updates(map[string]any{"status": "closed", "active_slot": nil, "closed_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// closePendingOrder replaces one unpaid order only after its live Alipay
// attempts have been closed. If Alipay reports that an attempt already paid,
// settlement wins and replacement is rejected to avoid duplicate entitlement.
func (s *Commerce) closePendingOrder(ctx context.Context, owner model.Owner, orderID uint64) error {
	type attemptRow struct {
		ID              uint64
		MerchantOrderNo string
	}
	var attempts []attemptRow
	if err := s.db.WithContext(ctx).Table("billing_payments").Select("id, merchant_order_no").
		Where("order_id = ? AND payment_environment = ? AND status IN ('created','pending')", orderID, s.Environment()).
		Order("created_at DESC").Find(&attempts).Error; err != nil {
		return err
	}
	for _, attempt := range attempts {
		if strings.TrimSpace(attempt.MerchantOrderNo) == "" {
			continue
		}
		if err := s.alipay.Close(ctx, attempt.MerchantOrderNo, s.now()); err != nil {
			result, queryErr := s.alipay.Query(ctx, attempt.MerchantOrderNo, s.now())
			resolution, resolveErr := resolveAlipayCloseFailure(err, result, queryErr)
			if resolveErr != nil {
				return fmt.Errorf("close Alipay trade before replacing order: %w", resolveErr)
			}
			if resolution == alipayTradePaid {
				now := s.now().UTC()
				if settleErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
					return s.settlePaymentTx(ctx, tx, attempt.ID, result.TradeNo, result.AmountFen, now)
				}); settleErr != nil {
					return settleErr
				}
				return ErrBillingPaymentCompleted
			}
		}
	}
	now := s.now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("billing_payments").Where("order_id = ? AND status IN ('created','pending')", orderID).
			Updates(map[string]any{"status": "closed", "active_slot": nil, "closed_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		closed := tx.Table("billing_orders").Where("id = ? AND owner_type = ? AND owner_id = ? AND status IN ('pending','paying')", orderID, owner.Type, owner.ID).
			Updates(map[string]any{"status": "closed", "active_slot": nil, "closed_at": now, "updated_at": now})
		if closed.Error != nil {
			return closed.Error
		}
		if closed.RowsAffected != 1 {
			return ErrBillingOrderNotPayable
		}
		return nil
	})
}

func (s *Commerce) ListOrders(ctx context.Context, owner model.Owner, page, pageSize int32) ([]*pb.BillingOrderInfo, int64, error) {
	if err := s.closeExpiredOrders(ctx, &owner); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	base := s.db.WithContext(ctx).Table("billing_orders").Where("owner_type = ? AND owner_id = ?", owner.Type, owner.ID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		OrderNo, OrderType, ProductKey, ProductName, Currency, Status, Environment string
		PriceVersionID, AmountFen                                                  int64
		ExpiresAt                                                                  time.Time
		PaidAt                                                                     *time.Time
		CreatedAt                                                                  time.Time
	}
	var rows []row
	err := s.db.WithContext(ctx).Raw(`SELECT orders.order_no, orders.order_type, product.product_key, product.name product_name,
		orders.price_version_id, orders.amount_fen, orders.currency, orders.status, orders.payment_environment environment,
		orders.expires_at, orders.paid_at, orders.created_at
		FROM billing_orders orders JOIN billing_products product ON product.id = orders.product_id
		WHERE orders.owner_type = ? AND orders.owner_id = ? ORDER BY orders.created_at DESC LIMIT ? OFFSET ?`, owner.Type, owner.ID, pageSize, (page-1)*pageSize).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	result := make([]*pb.BillingOrderInfo, 0, len(rows))
	for _, row := range rows {
		result = append(result, orderProto(row.OrderNo, row.OrderType, row.ProductKey, row.ProductName, row.Currency, row.Status, row.Environment, row.PriceVersionID, row.AmountFen, row.ExpiresAt, row.PaidAt, row.CreatedAt))
	}
	return result, total, nil
}

func (s *Commerce) CreateOrder(ctx context.Context, owner model.Owner, actorUserID, priceVersionID uint64, orderType, idempotencyKey string, replacePendingOrder bool) (*pb.BillingOrderInfo, error) {
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	if actorUserID == 0 || priceVersionID == 0 || strings.TrimSpace(idempotencyKey) == "" {
		return nil, errors.New("actor, price and idempotency key are required")
	}
	if orderType != "subscribe" && orderType != "renew" && orderType != "upgrade" && orderType != "credit_pack" {
		return nil, errors.New("invalid billing order type")
	}
	now := s.now().UTC()
	if err := s.closeExpiredOrders(ctx, &owner); err != nil {
		return nil, err
	}
	if existing, err := s.getOrderByIdempotency(ctx, owner, idempotencyKey); err == nil {
		if existing.PriceVersionId != int64(priceVersionID) || existing.OrderType != orderType {
			return nil, errors.New("idempotency key was already used for another billing order")
		}
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	type priceRow struct {
		ProductID                                                              uint64
		ProductKey, ProductName, ProductType, OwnerType, BillingTerm, Currency string
		AmountFen                                                              uint64
	}
	var price priceRow
	if err := s.db.WithContext(ctx).Raw(`SELECT product.id product_id, product.product_key, product.name product_name, product.product_type, product.owner_type,
		price.billing_term, price.amount_fen, price.currency FROM billing_price_versions price JOIN billing_products product ON product.id = price.product_id
		WHERE price.id = ? AND price.status = 'published' AND product.status = 'active' AND price.effective_at <= UTC_TIMESTAMP(3)
		  AND (price.retired_at IS NULL OR price.retired_at > UTC_TIMESTAMP(3))`, priceVersionID).Scan(&price).Error; err != nil {
		return nil, err
	}
	if price.ProductID == 0 || price.OwnerType != string(owner.Type) {
		return nil, errors.New("billing price is not available for this owner")
	}
	if price.AmountFen == 0 {
		return nil, errors.New("free products do not create payment orders")
	}
	if (orderType == "credit_pack") != (price.ProductType == "credit_pack") {
		return nil, errors.New("order type does not match product")
	}
	type openOrderRow struct {
		ID                                                                         uint64
		OrderNo, OrderType, ProductKey, ProductName, Currency, Status, Environment string
		PriceVersionID, AmountFen                                                  int64
		ExpiresAt, CreatedAt                                                       time.Time
	}
	var openOrder openOrderRow
	if err := s.db.WithContext(ctx).Raw(`SELECT orders.id, orders.order_no, orders.order_type, product.product_key, product.name product_name,
		orders.price_version_id, orders.amount_fen, orders.currency, orders.status, orders.payment_environment environment,
		orders.expires_at, orders.created_at
		FROM billing_orders orders JOIN billing_products product ON product.id = orders.product_id
		WHERE orders.owner_type = ? AND orders.owner_id = ? AND orders.status IN ('pending','paying') AND orders.expires_at > ?
		ORDER BY orders.created_at DESC LIMIT 1`, owner.Type, owner.ID, now).Scan(&openOrder).Error; err != nil {
		return nil, err
	}
	if openOrder.ID > 0 {
		if !replacePendingOrder {
			return nil, ErrPendingBillingOrder
		}
	}
	amountFen := price.AmountFen
	if price.ProductType == "subscription" {
		var current struct {
			ProductID   uint64
			BillingTerm string
			AmountFen   uint64
			Start       time.Time
			End         time.Time
		}
		if err := s.db.WithContext(ctx).Raw(`SELECT subscription.product_id, price.billing_term, price.amount_fen,
			subscription.current_period_start start, subscription.current_period_end end
			FROM billing_subscriptions subscription JOIN billing_price_versions price ON price.id = subscription.price_version_id
			WHERE subscription.owner_type = ? AND subscription.owner_id = ? AND subscription.status = 'active'
			ORDER BY subscription.current_period_end DESC LIMIT 1`, owner.Type, owner.ID).Scan(&current).Error; err != nil {
			return nil, err
		}
		switch orderType {
		case "subscribe":
			if current.ProductID > 0 && current.End.After(now) {
				return nil, errors.New("an active subscription already exists")
			}
		case "renew":
			if current.ProductID == 0 || current.ProductID != price.ProductID {
				return nil, errors.New("renewal must use the current subscription product")
			}
			var scheduledCount int64
			if err := s.db.WithContext(ctx).Table("billing_subscriptions").Where("owner_type = ? AND owner_id = ? AND status = 'pending' AND current_period_start >= ?", owner.Type, owner.ID, now).Count(&scheduledCount).Error; err != nil {
				return nil, err
			}
			if scheduledCount > 0 {
				return nil, ErrSubscriptionRenewalScheduled
			}
		case "upgrade":
			if current.ProductID == 0 || !current.End.After(now) {
				return nil, errors.New("an active subscription is required for upgrade")
			}
			if current.ProductID == price.ProductID || current.BillingTerm != price.BillingTerm || price.AmountFen <= current.AmountFen {
				return nil, errors.New("upgrade requires a higher-priced product with the same billing term")
			}
			period := current.End.Sub(current.Start)
			remaining := current.End.Sub(now)
			if period <= 0 || remaining <= 0 {
				return nil, errors.New("current subscription period is invalid")
			}
			amountFen = proratedCeil(price.AmountFen-current.AmountFen, current.Start, current.End, now)
			if amountFen == 0 {
				amountFen = 1
			}
		}
	}
	if openOrder.ID > 0 {
		if err := s.closePendingOrder(ctx, owner, openOrder.ID); err != nil {
			return nil, err
		}
	}
	orderNo := "B" + now.Format("20060102150405") + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:12])
	expiresAt := now.Add(billingOrderPaymentWindow)
	row := map[string]any{"order_no": orderNo, "owner_type": owner.Type, "owner_id": owner.ID, "order_type": orderType, "product_id": price.ProductID, "price_version_id": priceVersionID, "amount_fen": amountFen, "currency": price.Currency, "status": "pending", "active_slot": 1, "payment_environment": s.Environment(), "idempotency_key": idempotencyKey, "expires_at": expiresAt, "created_at": now, "updated_at": now}
	err := s.db.WithContext(ctx).Table("billing_orders").Create(row).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		if existing, findErr := s.getOrderByIdempotency(ctx, owner, idempotencyKey); findErr == nil {
			return existing, nil
		}
		return nil, ErrPendingBillingOrder
	}
	if err != nil {
		return nil, err
	}
	return orderProto(orderNo, orderType, price.ProductKey, price.ProductName, price.Currency, "pending", s.Environment(), int64(priceVersionID), int64(amountFen), expiresAt, nil, now), nil
}

func (s *Commerce) CreatePayment(ctx context.Context, owner model.Owner, orderNo, scene string) (string, string, bool, time.Time, error) {
	type row struct {
		ID                                        uint64
		OrderNo, Status, Environment, ProductName string
		AmountFen                                 uint64
		ExpiresAt                                 time.Time
	}
	var order row
	if err := s.db.WithContext(ctx).Raw(`SELECT orders.id, orders.order_no, orders.status, orders.payment_environment environment, orders.amount_fen, orders.expires_at, product.name product_name
		FROM billing_orders orders JOIN billing_products product ON product.id = orders.product_id
		WHERE orders.order_no = ? AND orders.owner_type = ? AND orders.owner_id = ?`, orderNo, owner.Type, owner.ID).Scan(&order).Error; err != nil {
		return "", "", false, time.Time{}, err
	}
	if order.ID == 0 || (order.Status != "pending" && order.Status != "paying") || order.Environment != s.Environment() {
		return "", "", false, time.Time{}, ErrBillingOrderNotPayable
	}
	if !order.ExpiresAt.After(s.now().UTC()) {
		if err := s.closeExpiredOrders(ctx, &owner); err != nil {
			return "", "", false, time.Time{}, err
		}
		return "", "", false, time.Time{}, ErrBillingOrderExpired
	}
	type existingAttemptRow struct {
		ID              uint64
		PaymentNo       string
		MerchantOrderNo string
		Scene           string
	}
	now := s.now().UTC()
	paymentNo := "P" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:24]
	sourceApp := "hr"
	if owner.Type == model.OwnerUser {
		sourceApp = "candidate"
	}
	create := map[string]any{"order_id": order.ID, "payment_no": paymentNo, "merchant_order_no": paymentNo, "channel": "alipay", "scene": scene, "source_app": sourceApp, "payment_environment": s.Environment(), "amount_fen": order.AmountFen, "currency": "CNY", "status": "pending", "active_slot": 1, "expires_at": order.ExpiresAt, "next_reconcile_at": now.Add(time.Minute), "created_at": now, "updated_at": now}
	var existing existingAttemptRow
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("billing_orders").Where("id = ?", order.ID).Select("id").Scan(&struct{ ID uint64 }{}).Error; err != nil {
			return err
		}
		if err := tx.Table("billing_payments").Select("id, payment_no, merchant_order_no, scene").Where("order_id = ? AND active_slot = 1", order.ID).Limit(1).Scan(&existing).Error; err != nil {
			return err
		}
		if existing.ID > 0 {
			return nil
		}
		if err := tx.Table("billing_payments").Create(create).Error; err != nil {
			return err
		}
		updated := tx.Table("billing_orders").Where("id = ? AND status IN ('pending','paying') AND expires_at > ?", order.ID, now).
			Updates(map[string]any{"status": "paying", "updated_at": now})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return errors.New("billing order changed while creating payment")
		}
		return nil
	}); err != nil {
		return "", "", false, time.Time{}, err
	}
	reused := existing.ID > 0
	if reused {
		paymentNo = existing.PaymentNo
		scene = existing.Scene
	}
	redirectURL, err := s.alipay.PayURL(payment.PayRequest{OrderNo: paymentNo, Subject: order.ProductName, Scene: scene, AmountFen: order.AmountFen, ExpiresAt: order.ExpiresAt}, now)
	if err != nil {
		return "", "", reused, order.ExpiresAt, err
	}
	return paymentNo, redirectURL, reused, order.ExpiresAt, nil
}

// SyncPayment confirms an Alipay sandbox payment for return_url / resume flows.
// It queries the channel trade and settles locally when TRADE_SUCCESS/FINISHED.
func (s *Commerce) SyncPayment(ctx context.Context, owner model.Owner, orderNo string) (string, error) {
	type orderRow struct {
		ID          uint64
		Status      string
		Environment string `gorm:"column:payment_environment"`
		AmountFen   uint64
	}
	var order orderRow
	if err := s.db.WithContext(ctx).Raw(`SELECT id, status, payment_environment, amount_fen
		FROM billing_orders WHERE order_no = ? AND owner_type = ? AND owner_id = ?`, orderNo, owner.Type, owner.ID).Scan(&order).Error; err != nil {
		return "", err
	}
	if order.ID == 0 || order.Environment != s.Environment() {
		return "", ErrBillingOrderNotPayable
	}
	type paymentRow struct {
		ID              uint64
		PaymentNo       string
		MerchantOrderNo string
		Status          string
		AmountFen       uint64
	}
	var attempt paymentRow
	if err := s.db.WithContext(ctx).Table("billing_payments").Select("id, payment_no, merchant_order_no, status, amount_fen").
		Where("order_id = ? AND payment_environment = ?", order.ID, s.Environment()).
		Order("created_at DESC").Limit(1).Scan(&attempt).Error; err != nil {
		return "", err
	}
	if order.Status == "paid" || attempt.Status == "succeeded" {
		if strings.TrimSpace(attempt.PaymentNo) != "" {
			return attempt.PaymentNo, nil
		}
		return "", nil
	}
	if (order.Status != "pending" && order.Status != "paying") || attempt.ID == 0 || strings.TrimSpace(attempt.MerchantOrderNo) == "" {
		return "", ErrBillingOrderNotPayable
	}
	result, err := s.alipay.Query(ctx, attempt.MerchantOrderNo, s.now())
	if err != nil {
		return "", err
	}
	if result.TradeStatus != "TRADE_SUCCESS" && result.TradeStatus != "TRADE_FINISHED" {
		return "", ErrBillingPaymentPending
	}
	if result.AmountFen != attempt.AmountFen || result.AmountFen != order.AmountFen {
		return "", errors.New("Alipay payment amount mismatch")
	}
	now := s.now().UTC()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.settlePaymentTx(ctx, tx, attempt.ID, result.TradeNo, result.AmountFen, now)
	}); err != nil {
		return "", err
	}
	return attempt.PaymentNo, nil
}

func (s *Commerce) ResolveAlipayReturn(ctx context.Context, fields map[string]string) (string, string, error) {
	if err := s.alipay.VerifyReturn(fields); err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrInvalidAlipayNotification, err)
	}
	var payment struct {
		ID        uint64
		SourceApp string
	}
	if err := s.db.WithContext(ctx).Table("billing_payments").Where("merchant_order_no = ? AND payment_environment = ?", fields["out_trade_no"], s.Environment()).Select("id, source_app").Scan(&payment).Error; err != nil {
		return "", "", err
	}
	if payment.ID == 0 || (payment.SourceApp != "hr" && payment.SourceApp != "candidate") {
		return "", "", fmt.Errorf("%w: return payment is unknown", ErrInvalidAlipayNotification)
	}
	token := strings.ReplaceAll(uuid.NewString(), "-", "") + strings.ReplaceAll(uuid.NewString(), "-", "")
	digest := sha256.Sum256([]byte(token))
	if err := s.db.WithContext(ctx).Table("billing_payments").Where("id = ?", payment.ID).Updates(map[string]any{"return_token_hash": hex.EncodeToString(digest[:]), "updated_at": s.now().UTC()}).Error; err != nil {
		return "", "", err
	}
	return payment.SourceApp, token, nil
}

func (s *Commerce) SyncPaymentReturn(ctx context.Context, owner model.Owner, token string) (string, error) {
	digest := sha256.Sum256([]byte(strings.TrimSpace(token)))
	var resolved struct {
		OrderNo   string
		PaymentID uint64
	}
	if err := s.db.WithContext(ctx).Raw(`SELECT orders.order_no, payment.id payment_id FROM billing_payments payment JOIN billing_orders orders ON orders.id = payment.order_id
		WHERE payment.return_token_hash = ? AND orders.owner_type = ? AND orders.owner_id = ? AND payment.payment_environment = ?`, hex.EncodeToString(digest[:]), owner.Type, owner.ID, s.Environment()).Scan(&resolved).Error; err != nil {
		return "", err
	}
	if resolved.OrderNo == "" {
		return "", ErrBillingOrderNotPayable
	}
	paymentNo, err := s.SyncPayment(ctx, owner, resolved.OrderNo)
	if err == nil {
		_ = s.db.WithContext(ctx).Table("billing_payments").Where("id = ?", resolved.PaymentID).Updates(map[string]any{"return_token_hash": nil, "updated_at": s.now().UTC()}).Error
	}
	return paymentNo, err
}

func (s *Commerce) settlePaymentTx(ctx context.Context, tx *gorm.DB, paymentID uint64, tradeNo string, paidAmountFen uint64, now time.Time) error {
	type paymentRow struct {
		ID, OrderID, AmountFen uint64
		Status, Currency       string
		Environment            string `gorm:"column:payment_environment"`
	}
	var paid paymentRow
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("billing_payments").Where("id = ?", paymentID).Scan(&paid).Error; err != nil {
		return err
	}
	if paid.ID == 0 || paid.AmountFen != paidAmountFen || paid.Currency != "CNY" || paid.Environment != s.Environment() {
		return errors.New("Alipay payment amount, currency or environment mismatch")
	}
	var order commerceOrderRow
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("billing_orders").Where("id = ?", paid.OrderID).Scan(&order).Error; err != nil {
		return err
	}
	if order.ID == 0 || order.AmountFen != paidAmountFen || order.Currency != "CNY" || order.Environment != s.Environment() {
		return errors.New("Alipay order amount, currency or environment mismatch")
	}
	if !isPaymentSettleable(paid.Status) {
		return errors.New("billing payment is not payable")
	}
	if err := tx.Table("billing_payments").Where("id = ?", paid.ID).Updates(map[string]any{"status": "succeeded", "active_slot": nil, "channel_trade_no": tradeNo, "paid_at": now, "next_reconcile_at": nil, "updated_at": now}).Error; err != nil {
		return err
	}
	if err := tx.Table("billing_payments").Where("order_id = ? AND id <> ? AND status = 'pending'", order.ID, paid.ID).
		Updates(map[string]any{"status": "closed", "active_slot": nil, "closed_at": now, "next_reconcile_at": nil, "updated_at": now}).Error; err != nil {
		return err
	}
	if order.Status == "paid" {
		if paid.Status != "succeeded" {
			platformobservability.DefaultMetrics.RecordBillingEvent("payment", "duplicate_success")
			payload, _ := json.Marshal(map[string]any{"order_id": order.ID, "payment_id": paid.ID, "trade_no": tradeNo, "payment_environment": s.Environment()})
			return tx.Table("event_outbox").Clauses(noOpConflict("idempotency_key", "idempotency_key")).Create(map[string]any{"event_id": uuid.NewString(), "event_type": "billing.payment.duplicate_success", "aggregate_type": "billing_order", "aggregate_id": order.ID, "routing_key": "billing.payment.duplicate_success", "producer": "billing-service", "idempotency_key": fmt.Sprintf("billing-order:%d:duplicate-payment:%d", order.ID, paid.ID), "payload": string(payload), "status": 0, "created_at": now, "updated_at": now}).Error
		}
		return nil
	}
	if order.Status != "pending" && order.Status != "paying" && order.Status != "closed" {
		return errors.New("billing order is not payable")
	}
	if err := tx.Table("billing_orders").Where("id = ?", order.ID).Updates(map[string]any{"status": "paid", "active_slot": nil, "paid_at": now, "closed_at": nil, "updated_at": now}).Error; err != nil {
		return err
	}
	return s.activateOrder(ctx, tx, order, now)
}

// MySQL requires at least one assignment after ON DUPLICATE KEY UPDATE. GORM
// cannot infer a primary column when Create receives a map, so DoNothing would
// otherwise emit a dangling UPDATE clause. Reassigning the conflict key is a
// deterministic no-op and keeps the surrounding settlement transaction safe.
func noOpConflict(noOpColumn string, conflictColumns ...string) clause.OnConflict {
	columns := make([]clause.Column, 0, len(conflictColumns))
	for _, column := range conflictColumns {
		columns = append(columns, clause.Column{Name: column})
	}
	return clause.OnConflict{
		Columns:   columns,
		DoUpdates: clause.AssignmentColumns([]string{noOpColumn}),
	}
}

func isPaymentSettleable(status string) bool {
	switch status {
	case "created", "pending", "closing", "unknown", "closed", "succeeded":
		return true
	default:
		return false
	}
}

func (s *Commerce) ProcessAlipayNotification(ctx context.Context, fields map[string]string) error {
	platformobservability.DefaultMetrics.RecordBillingEvent("alipay_webhook", "received")
	payload, _ := json.Marshal(sanitizedAlipayNotification(fields))
	digest := sha256.Sum256(payload)
	eventKey := fields["notify_id"]
	if eventKey == "" {
		eventKey = "digest:" + hex.EncodeToString(digest[:])
	}
	now := s.now().UTC()
	event := map[string]any{"channel": "alipay", "payment_environment": s.Environment(), "event_key": eventKey, "event_type": fields["notify_type"], "signature_verified": false, "payload_sha256": hex.EncodeToString(digest[:]), "payload": string(payload), "status": "received", "created_at": now}
	if err := s.db.WithContext(ctx).Table("billing_webhook_events").Create(event).Error; errors.Is(err, gorm.ErrDuplicatedKey) {
		var existing struct{ PayloadSHA256, Status string }
		if queryErr := s.db.WithContext(ctx).Table("billing_webhook_events").Select("payload_sha256, status").Where("channel = 'alipay' AND payment_environment = ? AND event_key = ?", s.Environment(), eventKey).Scan(&existing).Error; queryErr != nil {
			return queryErr
		}
		if existing.PayloadSHA256 != hex.EncodeToString(digest[:]) {
			return fmt.Errorf("%w: duplicate notify_id has a different payload", ErrInvalidAlipayNotification)
		}
		if existing.Status == "processed" || existing.Status == "ignored" {
			return nil
		}
	} else if err != nil {
		return err
	}
	if err := s.alipay.VerifyNotification(fields); err != nil {
		platformobservability.DefaultMetrics.RecordBillingEvent("alipay_webhook", "signature_failed")
		_ = s.markWebhookFailed(ctx, eventKey, err)
		return fmt.Errorf("%w: %v", ErrInvalidAlipayNotification, err)
	}
	if err := s.db.WithContext(ctx).Table("billing_webhook_events").Where("channel = 'alipay' AND payment_environment = ? AND event_key = ?", s.Environment(), eventKey).Updates(map[string]any{"signature_verified": true, "status": "verified", "failure_reason": nil}).Error; err != nil {
		return err
	}
	tradeStatus := fields["trade_status"]
	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" && tradeStatus != "TRADE_CLOSED" {
		platformobservability.DefaultMetrics.RecordBillingEvent("alipay_webhook", "ignored")
		return s.db.WithContext(ctx).Table("billing_webhook_events").Where("channel = 'alipay' AND payment_environment = ? AND event_key = ?", s.Environment(), eventKey).Updates(map[string]any{"status": "ignored", "processed_at": now}).Error
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var paymentID uint64
		if err := tx.Table("billing_payments").Where("merchant_order_no = ? AND payment_environment = ?", fields["out_trade_no"], s.Environment()).Select("id").Scan(&paymentID).Error; err != nil {
			return err
		}
		if paymentID == 0 {
			return fmt.Errorf("%w: unknown payment", ErrInvalidAlipayNotification)
		}
		if tradeStatus == "TRADE_CLOSED" {
			closed := tx.Table("billing_payments").Where("id = ? AND active_slot = 1 AND status IN ('created','pending','closing','unknown')", paymentID).Updates(map[string]any{"status": "closed", "active_slot": nil, "closed_at": now, "updated_at": now})
			if closed.Error != nil {
				return closed.Error
			}
			if closed.RowsAffected == 0 {
				return tx.Table("billing_webhook_events").Where("channel = 'alipay' AND event_key = ? AND payment_environment = ?", eventKey, s.Environment()).Updates(map[string]any{"status": "ignored", "processed_at": now}).Error
			}
			return tx.Exec(`UPDATE billing_orders orders JOIN billing_payments payment ON payment.order_id = orders.id
				SET orders.status = 'closed', orders.active_slot = NULL, orders.closed_at = ?, orders.updated_at = ?
				WHERE payment.id = ? AND orders.status IN ('pending','paying')`, now, now, paymentID).Error
		}
		amountFen, err := parseAmountFen(fields["total_amount"])
		if err != nil {
			return fmt.Errorf("%w: invalid amount", ErrInvalidAlipayNotification)
		}
		if err := s.settlePaymentTx(ctx, tx, paymentID, fields["trade_no"], amountFen, now); err != nil {
			return err
		}
		return tx.Table("billing_webhook_events").Where("channel = 'alipay' AND event_key = ? AND payment_environment = ?", eventKey, s.Environment()).Updates(map[string]any{"status": "processed", "processed_at": now}).Error
	})
	if err != nil {
		platformobservability.DefaultMetrics.RecordBillingEvent("alipay_webhook", "processing_failed")
		_ = s.markWebhookFailed(ctx, eventKey, err)
	} else {
		platformobservability.DefaultMetrics.RecordBillingEvent("alipay_webhook", "processed")
	}
	return err
}

func (s *Commerce) markWebhookFailed(ctx context.Context, eventKey string, cause error) error {
	reason := cause.Error()
	if len(reason) > 500 {
		reason = reason[:500]
	}
	return s.db.WithContext(ctx).Table("billing_webhook_events").Where("channel = 'alipay' AND payment_environment = ? AND event_key = ?", s.Environment(), eventKey).Updates(map[string]any{"status": "failed", "failure_reason": reason, "retry_count": gorm.Expr("retry_count + 1"), "last_attempt_at": s.now().UTC()}).Error
}

func sanitizedAlipayNotification(fields map[string]string) map[string]string {
	allowed := map[string]struct{}{"notify_id": {}, "notify_type": {}, "notify_time": {}, "app_id": {}, "seller_id": {}, "out_trade_no": {}, "trade_no": {}, "trade_status": {}, "total_amount": {}, "receipt_amount": {}, "buyer_pay_amount": {}, "gmt_create": {}, "gmt_payment": {}, "gmt_close": {}}
	result := make(map[string]string, len(allowed))
	for key := range allowed {
		if value := strings.TrimSpace(fields[key]); value != "" {
			result[key] = value
		}
	}
	return result
}

func (s *Commerce) activateOrder(ctx context.Context, tx *gorm.DB, order commerceOrderRow, now time.Time) error {
	type priceRow struct {
		BillingTerm, ProductType string
		IncludedCredits          uint64
	}
	var price priceRow
	if err := tx.Raw(`SELECT price.billing_term, price.included_credits, product.product_type FROM billing_price_versions price JOIN billing_products product ON product.id = price.product_id WHERE price.id = ?`, order.PriceVersionID).Scan(&price).Error; err != nil {
		return err
	}
	owner := model.Owner{Type: model.OwnerType(order.OwnerType), ID: order.OwnerID}
	if price.ProductType == "credit_pack" {
		return createCreditGrant(tx, owner, "order", order.ID, "credit_pack", price.IncludedCredits, now, now.AddDate(1, 0, 0))
	}
	var current struct {
		ID              uint64
		Start, End      time.Time
		IncludedCredits uint64
	}
	if err := tx.Raw(`SELECT subscription.id, subscription.current_period_start start, subscription.current_period_end end,
		price.included_credits FROM billing_subscriptions subscription
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = ? AND subscription.owner_id = ? AND subscription.status = 'active'
		ORDER BY subscription.current_period_end DESC LIMIT 1`, owner.Type, owner.ID).Scan(&current).Error; err != nil {
		return err
	}
	start := now
	if order.OrderType == "renew" && current.ID > 0 && current.End.After(now) {
		start = current.End
	}
	end := start.AddDate(0, 1, 0)
	if price.BillingTerm == "yearly" {
		end = start.AddDate(1, 0, 0)
	}
	if order.OrderType == "upgrade" && current.ID > 0 && current.End.After(now) {
		end = current.End
	}
	subscriptionStatus := "active"
	if order.OrderType == "renew" && start.After(now) {
		subscriptionStatus = "pending"
	} else if current.ID > 0 {
		if err := tx.Table("billing_subscriptions").Where("id = ?", current.ID).Updates(map[string]any{"status": "expired", "updated_at": now}).Error; err != nil {
			return err
		}
	}
	subscription := map[string]any{"owner_type": owner.Type, "owner_id": owner.ID, "product_id": order.ProductID, "price_version_id": order.PriceVersionID, "status": subscriptionStatus, "term": price.BillingTerm, "current_period_start": start, "current_period_end": end, "activated_by_order_id": order.ID, "created_at": now, "updated_at": now}
	if err := tx.Table("billing_subscriptions").Create(subscription).Error; err != nil {
		return err
	}
	var subscriptionID uint64
	if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&subscriptionID).Error; err != nil {
		return err
	}
	if err := tx.Table("billing_orders").Where("id = ?", order.ID).Updates(map[string]any{"subscription_id": subscriptionID, "updated_at": now}).Error; err != nil {
		return err
	}
	if subscriptionStatus == "pending" {
		eventPayload, _ := json.Marshal(map[string]any{"subscription_id": subscriptionID, "owner_type": owner.Type, "owner_id": owner.ID, "price_version_id": order.PriceVersionID, "period_start": start, "period_end": end, "environment": s.Environment()})
		return tx.Table("event_outbox").Create(map[string]any{"event_id": uuid.NewString(), "event_type": "billing.subscription.scheduled", "aggregate_type": "billing_subscription", "aggregate_id": subscriptionID, "routing_key": "billing.subscription.scheduled", "producer": "billing-service", "idempotency_key": fmt.Sprintf("billing-subscription:%d:scheduled", subscriptionID), "payload": string(eventPayload), "status": 0, "created_at": now, "updated_at": now}).Error
	}
	grantCredits := price.IncludedCredits
	if order.OrderType == "upgrade" && current.End.After(now) {
		period := current.End.Sub(current.Start)
		remaining := current.End.Sub(now)
		additionalCredits := uint64(0)
		if price.IncludedCredits > current.IncludedCredits {
			additionalCredits = price.IncludedCredits - current.IncludedCredits
		}
		if period > 0 && remaining > 0 {
			grantCredits = proratedCeil(additionalCredits, current.Start, current.End, now)
		}
	}
	monthStart, monthEnd := billingMonth(now)
	if end.Before(monthEnd) {
		monthEnd = end
	}
	if err := createCreditGrant(tx, owner, "subscription", subscriptionID, "subscription_monthly", grantCredits, monthStart, monthEnd); err != nil {
		return err
	}
	eventPayload, _ := json.Marshal(map[string]any{"subscription_id": subscriptionID, "owner_type": owner.Type, "owner_id": owner.ID, "price_version_id": order.PriceVersionID, "period_start": start, "period_end": end, "environment": s.Environment()})
	return tx.Table("event_outbox").Create(map[string]any{"event_id": uuid.NewString(), "event_type": "billing.subscription.activated", "aggregate_type": "billing_subscription", "aggregate_id": subscriptionID, "routing_key": "billing.subscription.activated", "producer": "billing-service", "idempotency_key": fmt.Sprintf("billing-subscription:%d:activated", subscriptionID), "payload": string(eventPayload), "status": 0, "created_at": now, "updated_at": now}).Error
}

func (s *Commerce) RequestRefund(ctx context.Context, owner model.Owner, actorUserID uint64, orderNo, reason, idempotencyKey string) (string, string, string, error) {
	if actorUserID == 0 || strings.TrimSpace(reason) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return "", "", "", errors.New("refund actor and reason are required")
	}
	type row struct {
		ID, PaymentID, AmountFen uint64
		OrderType, Status        string
		MerchantOrderNo          string
		PaidAt                   time.Time
	}
	var order row
	if err := s.db.WithContext(ctx).Raw(`SELECT orders.id, orders.order_type, orders.status, orders.amount_fen, orders.paid_at,
		payment.id payment_id, payment.merchant_order_no
		FROM billing_orders orders JOIN billing_payments payment ON payment.order_id = orders.id AND payment.status = 'succeeded'
		WHERE orders.order_no = ? AND orders.owner_type = ? AND orders.owner_id = ?`, orderNo, owner.Type, owner.ID).Scan(&order).Error; err != nil {
		return "", "", "", err
	}
	if order.ID == 0 {
		return "", "", "", errors.New("only paid orders can be refunded")
	}
	var existing struct{ RefundNo, Status, ReviewMode string }
	if err := s.db.WithContext(ctx).Table("billing_refunds").Select("refund_no, status, review_mode").Where("order_id = ? AND idempotency_key = ?", order.ID, idempotencyKey).Scan(&existing).Error; err != nil {
		return "", "", "", err
	}
	if existing.RefundNo != "" {
		return existing.RefundNo, existing.Status, existing.ReviewMode, nil
	}
	if order.Status != "paid" {
		return "", "", "", errors.New("only paid orders without an active refund can be refunded")
	}
	var usageCount int64
	if err := s.db.WithContext(ctx).Table("ai_usage_events").Where("owner_type = ? AND owner_id = ? AND occurred_at >= ? AND credits_charged > 0", owner.Type, owner.ID, order.PaidAt).Count(&usageCount).Error; err != nil {
		return "", "", "", err
	}
	reviewMode := "automatic"
	statusValue := "processing"
	if usageCount > 0 || order.OrderType == "renew" || order.OrderType == "upgrade" {
		reviewMode = "manual"
		statusValue = "reviewing"
	}
	refundNo := "R" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:24]
	now := s.now().UTC()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("billing_refunds").Create(map[string]any{"refund_no": refundNo, "idempotency_key": idempotencyKey, "order_id": order.ID, "payment_id": order.PaymentID, "amount_fen": order.AmountFen, "reason": reason, "status": statusValue, "active_slot": 1, "review_mode": reviewMode, "requested_by": actorUserID, "next_reconcile_at": func() any {
			if statusValue == "processing" {
				return now.Add(time.Minute)
			}
			return nil
		}(), "created_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Table("billing_orders").Where("id = ? AND status = 'paid'", order.ID).Updates(map[string]any{"status": "refunding", "updated_at": now}).Error
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			if queryErr := s.db.WithContext(ctx).Table("billing_refunds").Select("refund_no, status, review_mode").Where("order_id = ? AND idempotency_key = ?", order.ID, idempotencyKey).Scan(&existing).Error; queryErr == nil && existing.RefundNo != "" {
				return existing.RefundNo, existing.Status, existing.ReviewMode, nil
			}
		}
		return "", "", "", err
	}
	if reviewMode == "manual" {
		platformobservability.DefaultMetrics.RecordBillingEvent("refund", "reviewing")
		return refundNo, statusValue, reviewMode, nil
	}
	channelNo, err := s.alipay.Refund(ctx, order.MerchantOrderNo, refundNo, reason, order.AmountFen, now)
	if err != nil {
		if payment.IsAPIRejected(err) {
			if failErr := s.failRefund(ctx, refundNo, err, now); failErr != nil {
				return refundNo, "failed", reviewMode, failErr
			}
			platformobservability.DefaultMetrics.RecordBillingEvent("refund", "failed")
			return refundNo, "failed", reviewMode, nil
		}
		platformobservability.DefaultMetrics.RecordBillingEvent("refund", "unknown")
		_ = s.db.WithContext(ctx).Table("billing_refunds").Where("refund_no = ?", refundNo).Updates(map[string]any{"status": "unknown", "next_reconcile_at": now.Add(time.Minute), "last_error": truncateError(err), "updated_at": now}).Error
		return refundNo, "unknown", reviewMode, nil
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("billing_refunds").Where("refund_no = ?", refundNo).Updates(map[string]any{"status": "succeeded", "active_slot": nil, "channel_refund_no": channelNo, "succeeded_at": now, "next_reconcile_at": nil, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Table("billing_orders").Where("id = ?", order.ID).Update("status", "refunded").Error; err != nil {
			return err
		}
		if err := tx.Table("billing_payments").Where("id = ?", order.PaymentID).Update("status", "refunded").Error; err != nil {
			return err
		}
		var subscriptionIDs []uint64
		if err := tx.Table("billing_subscriptions").Where("activated_by_order_id = ?", order.ID).Pluck("id", &subscriptionIDs).Error; err != nil {
			return err
		}
		var grants []struct{ ID, RemainingCredits uint64 }
		grantQuery := tx.Table("ai_credit_grants").Where("owner_type = ? AND owner_id = ? AND ((source_type = 'order' AND source_id = ?) OR (source_type = 'subscription' AND source_id IN ?)) AND status = 'active'", owner.Type, owner.ID, order.ID, subscriptionIDs)
		if err := grantQuery.Find(&grants).Error; err != nil {
			return err
		}
		var balance uint64
		if err := tx.Table("ai_credit_grants").Where("owner_type = ? AND owner_id = ? AND status = 'active' AND valid_from <= ? AND (expires_at IS NULL OR expires_at > ?)", owner.Type, owner.ID, now, now).Select("COALESCE(SUM(remaining_credits), 0)").Scan(&balance).Error; err != nil {
			return err
		}
		for _, grant := range grants {
			if grant.RemainingCredits > 0 {
				if grant.RemainingCredits <= balance {
					balance -= grant.RemainingCredits
				} else {
					balance = 0
				}
				if err := tx.Table("ai_credit_ledger").Create(map[string]any{"entry_id": uuid.NewString(), "owner_type": owner.Type, "owner_id": owner.ID, "grant_id": grant.ID, "entry_type": "refund", "credits_delta": -int64(grant.RemainingCredits), "balance_after": int64(balance), "idempotency_key": fmt.Sprintf("refund:%s:grant:%d", refundNo, grant.ID), "description": reason, "created_at": now}).Error; err != nil {
					return err
				}
			}
		}
		if err := grantQuery.Updates(map[string]any{"status": "revoked", "remaining_credits": 0, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Table("billing_subscriptions").Where("activated_by_order_id = ? AND status = 'active'", order.ID).Updates(map[string]any{"status": "cancelled", "updated_at": now}).Error
	})
	if err != nil {
		return refundNo, "failed", reviewMode, err
	}
	return refundNo, "succeeded", reviewMode, nil
}

func (s *Commerce) UpdateOperationalMetrics(ctx context.Context) error {
	type row struct {
		Status string
		Count  int64
	}
	var payments []row
	for _, statusValue := range []string{"pending", "closing", "unknown"} {
		platformobservability.DefaultMetrics.SetBillingGauge("payment", statusValue, 0)
	}
	if err := s.db.WithContext(ctx).Table("billing_payments").Select("status, COUNT(*) count").Where("status IN ('pending','closing','unknown')").Group("status").Scan(&payments).Error; err != nil {
		return err
	}
	for _, item := range payments {
		platformobservability.DefaultMetrics.SetBillingGauge("payment", item.Status, float64(item.Count))
	}
	var refunds []row
	for _, statusValue := range []string{"reviewing", "processing", "unknown"} {
		platformobservability.DefaultMetrics.SetBillingGauge("refund", statusValue, 0)
	}
	if err := s.db.WithContext(ctx).Table("billing_refunds").Select("status, COUNT(*) count").Where("status IN ('reviewing','processing','unknown')").Group("status").Scan(&refunds).Error; err != nil {
		return err
	}
	for _, item := range refunds {
		platformobservability.DefaultMetrics.SetBillingGauge("refund", item.Status, float64(item.Count))
	}
	var oldestSeconds float64
	if err := s.db.WithContext(ctx).Raw(`SELECT COALESCE(TIMESTAMPDIFF(SECOND, MIN(created_at), UTC_TIMESTAMP(3)), 0) FROM billing_payments WHERE status IN ('pending','closing','unknown')`).Scan(&oldestSeconds).Error; err != nil {
		return err
	}
	platformobservability.DefaultMetrics.SetBillingGauge("payment", "oldest_age_seconds", oldestSeconds)
	return nil
}

func (s *Commerce) ListRefunds(ctx context.Context, statusFilter string, page, pageSize int32) ([]*pb.BillingRefundInfo, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := s.db.WithContext(ctx).Table("billing_refunds refund")
	if statusFilter != "" {
		query = query.Where("refund.status = ?", statusFilter)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		RefundNo, OrderNo, OwnerType, Status, ReviewMode, Reason, LastError string
		OwnerID, AmountFen, RequestedBy                                     uint64
		ReviewedBy                                                          *uint64
		CreatedAt                                                           time.Time
		ReviewedAt                                                          *time.Time
	}
	var rows []row
	base := `SELECT refund.refund_no, orders.order_no, orders.owner_type, orders.owner_id, refund.amount_fen, refund.status, refund.review_mode,
		refund.reason, refund.requested_by, refund.reviewed_by, refund.created_at, refund.reviewed_at, COALESCE(refund.last_error, '') last_error
		FROM billing_refunds refund JOIN billing_orders orders ON orders.id = refund.order_id`
	args := []any{}
	if statusFilter != "" {
		base += " WHERE refund.status = ?"
		args = append(args, statusFilter)
	}
	base += " ORDER BY refund.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)
	if err := s.db.WithContext(ctx).Raw(base, args...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*pb.BillingRefundInfo, 0, len(rows))
	for _, row := range rows {
		item := &pb.BillingRefundInfo{RefundNo: row.RefundNo, OrderNo: row.OrderNo, OwnerType: row.OwnerType, OwnerId: int64(row.OwnerID), AmountFen: int64(row.AmountFen), Status: row.Status, ReviewMode: row.ReviewMode, Reason: row.Reason, RequestedBy: int64(row.RequestedBy), CreatedAtUnixMs: row.CreatedAt.UnixMilli(), LastError: row.LastError}
		if row.ReviewedBy != nil {
			item.ReviewedBy = int64(*row.ReviewedBy)
		}
		if row.ReviewedAt != nil {
			item.ReviewedAtUnixMs = row.ReviewedAt.UnixMilli()
		}
		result = append(result, item)
	}
	return result, total, nil
}

func (s *Commerce) ReviewRefund(ctx context.Context, refundNo string, actorUserID uint64, action, reason string) (string, string, error) {
	if refundNo == "" || actorUserID == 0 || (action != "approve" && action != "reject") {
		return "", "", errors.New("refund, actor and valid action are required")
	}
	type row struct {
		OrderID, AmountFen                    uint64
		Status, MerchantOrderNo, RefundReason string
	}
	var refund row
	if err := s.db.WithContext(ctx).Raw(`SELECT refund.order_id, refund.amount_fen, refund.status, refund.reason refund_reason, payment.merchant_order_no
		FROM billing_refunds refund JOIN billing_payments payment ON payment.id = refund.payment_id WHERE refund.refund_no = ?`, refundNo).Scan(&refund).Error; err != nil {
		return "", "", err
	}
	if refund.OrderID == 0 || refund.Status != "reviewing" {
		return "", "", errors.New("refund is not awaiting review")
	}
	now := s.now().UTC()
	if action == "reject" {
		if strings.TrimSpace(reason) == "" {
			return "", "", errors.New("rejection reason is required")
		}
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			updated := tx.Table("billing_refunds").Where("refund_no = ? AND status = 'reviewing'", refundNo).Updates(map[string]any{"status": "rejected", "active_slot": nil, "reviewed_by": actorUserID, "reviewed_at": now, "rejected_reason": reason, "updated_at": now})
			if updated.Error != nil {
				return updated.Error
			}
			if updated.RowsAffected != 1 {
				return errors.New("refund review state changed")
			}
			return tx.Table("billing_orders").Where("id = ? AND status = 'refunding'", refund.OrderID).Updates(map[string]any{"status": "paid", "updated_at": now}).Error
		})
		return refundNo, "rejected", err
	}
	updated := s.db.WithContext(ctx).Table("billing_refunds").Where("refund_no = ? AND status = 'reviewing'", refundNo).Updates(map[string]any{"status": "processing", "reviewed_by": actorUserID, "reviewed_at": now, "next_reconcile_at": now.Add(time.Minute), "updated_at": now})
	if updated.Error != nil {
		return "", "", updated.Error
	}
	if updated.RowsAffected != 1 {
		return "", "", errors.New("refund review state changed")
	}
	channelNo, err := s.alipay.Refund(ctx, refund.MerchantOrderNo, refundNo, refund.RefundReason, refund.AmountFen, now)
	if err != nil {
		if payment.IsAPIRejected(err) {
			if failErr := s.failRefund(ctx, refundNo, err, now); failErr != nil {
				return refundNo, "failed", failErr
			}
			return refundNo, "failed", nil
		}
		_ = s.db.WithContext(ctx).Table("billing_refunds").Where("refund_no = ?", refundNo).Updates(map[string]any{"status": "unknown", "last_error": truncateError(err), "next_reconcile_at": now.Add(time.Minute), "updated_at": now}).Error
		return refundNo, "unknown", nil
	}
	if err := s.finalizeRefund(ctx, refundNo, channelNo, now); err != nil {
		return refundNo, "processing", err
	}
	return refundNo, "succeeded", nil
}

func truncateError(err error) string {
	if err == nil {
		return ""
	}
	value := err.Error()
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

func createCreditGrant(tx *gorm.DB, owner model.Owner, sourceType string, sourceID uint64, grantType string, credits uint64, validFrom, expiresAt time.Time) error {
	if credits == 0 {
		return nil
	}
	now := time.Now().UTC()
	if err := tx.Table("ai_credit_grants").Create(map[string]any{"owner_type": owner.Type, "owner_id": owner.ID, "grant_type": grantType, "source_type": sourceType, "source_id": sourceID, "total_credits": credits, "remaining_credits": credits, "valid_from": validFrom, "expires_at": expiresAt, "status": "active", "created_at": now, "updated_at": now}).Error; err != nil {
		return err
	}
	var grantID uint64
	if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&grantID).Error; err != nil {
		return err
	}
	var balance uint64
	if err := tx.Table("ai_credit_grants").Where("owner_type = ? AND owner_id = ? AND status = 'active' AND valid_from <= ? AND (expires_at IS NULL OR expires_at > ?)", owner.Type, owner.ID, now, now).Select("COALESCE(SUM(remaining_credits), 0)").Scan(&balance).Error; err != nil {
		return err
	}
	return tx.Table("ai_credit_ledger").Create(map[string]any{"entry_id": uuid.NewString(), "owner_type": owner.Type, "owner_id": owner.ID, "grant_id": grantID, "entry_type": "grant", "credits_delta": int64(credits), "balance_after": int64(balance), "idempotency_key": fmt.Sprintf("grant:%s:%d:%s", sourceType, sourceID, validFrom.Format(time.RFC3339)), "description": "paid AI credit grant", "created_at": now}).Error
}

func (s *Commerce) getOrderByIdempotency(ctx context.Context, owner model.Owner, key string) (*pb.BillingOrderInfo, error) {
	type row struct {
		OrderNo, OrderType, ProductKey, ProductName, Currency, Status, Environment string
		PriceVersionID, AmountFen                                                  int64
		ExpiresAt                                                                  time.Time
		PaidAt                                                                     *time.Time
		CreatedAt                                                                  time.Time
	}
	var existing row
	if err := s.db.WithContext(ctx).Raw(`SELECT orders.order_no, orders.order_type, product.product_key, product.name product_name,
		orders.price_version_id, orders.amount_fen, orders.currency, orders.status, orders.payment_environment environment,
		orders.expires_at, orders.paid_at, orders.created_at
		FROM billing_orders orders JOIN billing_products product ON product.id = orders.product_id
		WHERE orders.owner_type = ? AND orders.owner_id = ? AND orders.idempotency_key = ? LIMIT 1`, owner.Type, owner.ID, key).Scan(&existing).Error; err != nil {
		return nil, err
	}
	if existing.OrderNo == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return orderProto(existing.OrderNo, existing.OrderType, existing.ProductKey, existing.ProductName, existing.Currency,
		existing.Status, existing.Environment, existing.PriceVersionID, existing.AmountFen, existing.ExpiresAt, existing.PaidAt, existing.CreatedAt), nil
}

func orderProto(orderNo, orderType, productKey, productName, currency, statusValue, environment string, priceVersionID, amountFen int64, expiresAt time.Time, paidAt *time.Time, createdAt time.Time) *pb.BillingOrderInfo {
	result := &pb.BillingOrderInfo{OrderNo: orderNo, OrderType: orderType, ProductKey: productKey, ProductName: productName, PriceVersionId: priceVersionID, AmountFen: amountFen, Currency: currency, Status: statusValue, PaymentEnvironment: environment, ExpiresAtUnixMs: expiresAt.UnixMilli(), CreatedAtUnixMs: createdAt.UnixMilli()}
	if paidAt != nil {
		result.PaidAtUnixMs = paidAt.UnixMilli()
	}
	return result
}

func parseAmountFen(value string) (uint64, error) {
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts) == 0 {
		return 0, errors.New("invalid amount")
	}
	yuan, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	fen := uint64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
		if len(fraction) != 2 {
			return 0, errors.New("amount must have at most two decimals")
		}
		fen, err = strconv.ParseUint(fraction, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	return yuan*100 + fen, nil
}

func billingMonth(now time.Time) (time.Time, time.Time) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
	return start.UTC(), start.AddDate(0, 1, 0).UTC()
}

func proratedCeil(value uint64, periodStart, periodEnd, now time.Time) uint64 {
	period := periodEnd.Sub(periodStart)
	remaining := periodEnd.Sub(now)
	if value == 0 || period <= 0 || remaining <= 0 {
		return 0
	}
	if remaining >= period {
		return value
	}
	return uint64(math.Ceil(float64(value) * remaining.Seconds() / period.Seconds()))
}

func ptrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func ptrInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
