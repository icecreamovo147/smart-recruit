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
	"smart-recruit-proto/recruitment/pb"
)

type AlipayGateway interface {
	Environment() string
	PayURL(payment.PayRequest, time.Time) (string, error)
	VerifyNotification(map[string]string) error
	Refund(context.Context, string, string, string, uint64, time.Time) (string, error)
	Query(context.Context, string, time.Time) (payment.QueryResult, error)
}

func (s *Commerce) ReconcilePendingPayments(ctx context.Context, limit int) error {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var orders []commerceOrderRow
	if err := s.db.WithContext(ctx).Table("billing_orders").Where("status = 'paying' AND payment_environment = ?", s.Environment()).Order("created_at").Limit(limit).Find(&orders).Error; err != nil {
		return err
	}
	for _, order := range orders {
		result, err := s.alipay.Query(ctx, order.OrderNo, s.now())
		if err != nil {
			continue
		}
		if result.TradeStatus != "TRADE_SUCCESS" && result.TradeStatus != "TRADE_FINISHED" {
			continue
		}
		if result.AmountFen != order.AmountFen {
			continue
		}
		now := s.now().UTC()
		payload, _ := json.Marshal(result)
		digest := sha256.Sum256(payload)
		eventKey := "query:" + result.TradeNo + ":" + result.TradeStatus
		_ = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var locked commerceOrderRow
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("billing_orders").Where("id = ?", order.ID).Scan(&locked).Error; err != nil {
				return err
			}
			if locked.Status == "paid" {
				return nil
			}
			if locked.Status != "paying" {
				return nil
			}
			if err := tx.Table("billing_webhook_events").Clauses(clause.OnConflict{DoNothing: true}).Create(map[string]any{"channel": "alipay", "payment_environment": s.Environment(), "event_key": eventKey, "event_type": "active_query", "signature_verified": false, "payload_sha256": hex.EncodeToString(digest[:]), "payload": string(payload), "status": "processed", "processed_at": now, "created_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Table("billing_payments").Where("order_id = ? AND payment_environment = ?", locked.ID, s.Environment()).Updates(map[string]any{"status": "succeeded", "channel_trade_no": result.TradeNo, "paid_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Table("billing_orders").Where("id = ?", locked.ID).Updates(map[string]any{"status": "paid", "paid_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
			return s.activateOrder(ctx, tx, locked, now)
		})
	}
	return s.db.WithContext(ctx).Table("billing_orders").Where("status IN ('pending','paying') AND expires_at <= ?", s.now().UTC()).Updates(map[string]any{"status": "closed", "closed_at": s.now().UTC(), "updated_at": s.now().UTC()}).Error
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
	now := s.now().UTC()
	var result pb.BillingPriceInfo
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var productType string
		if err := tx.Table("billing_products").Where("id = ?", productID).Select("product_type").Scan(&productType).Error; err != nil {
			return err
		}
		if productType == "" {
			return gorm.ErrRecordNotFound
		}
		if productType == "credit_pack" && term != "one_time" {
			return errors.New("credit packs must use one_time billing")
		}
		if productType == "subscription" && term == "one_time" {
			return errors.New("subscriptions must use monthly or yearly billing")
		}
		statusValue := "draft"
		var effectiveAt any = nil
		if publish {
			statusValue = "published"
			effectiveAt = now
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
			if err := tx.Table("billing_price_versions").Where("id = ?", priceVersionID).Updates(map[string]any{"billing_term": term, "amount_fen": amountFen, "included_credits": credits, "entitlement_snapshot": snapshot, "status": statusValue, "effective_at": effectiveAt, "updated_at": now}).Error; err != nil {
				return err
			}
			result.Id = int64(priceVersionID)
			result.Version = existing.Version
		} else {
			var version int32
			if err := tx.Table("billing_price_versions").Where("product_id = ?", productID).Select("COALESCE(MAX(version), 0) + 1").Scan(&version).Error; err != nil {
				return err
			}
			row := map[string]any{"product_id": productID, "version": version, "billing_term": term, "amount_fen": amountFen, "currency": "CNY", "included_credits": credits, "entitlement_snapshot": snapshot, "status": statusValue, "effective_at": effectiveAt, "created_at": now, "updated_at": now}
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
			if err := tx.Table("billing_price_versions").Where("product_id = ? AND billing_term = ? AND id <> ? AND status = 'published'", productID, term, result.Id).Updates(map[string]any{"status": "retired", "retired_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Table("billing_products").Where("id = ?", productID).Updates(map[string]any{"status": "active", "updated_at": now}).Error; err != nil {
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
		effectiveAt = now
	}
	var result pb.AIRateCardInfo
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var version int32
		if err := tx.Table("ai_rate_cards").Where("provider_key = ? AND model_key = ?", providerKey, modelKey).Select("COALESCE(MAX(version), 0) + 1").Scan(&version).Error; err != nil {
			return err
		}
		row := map[string]any{"provider_key": providerKey, "model_key": modelKey, "version": version, "currency": "CNY", "input_micros_per_1k_tokens": input, "output_micros_per_1k_tokens": output, "cached_input_micros_per_1k_tokens": cached, "credit_micros": credit, "status": statusValue, "effective_at": effectiveAt, "created_at": now}
		if err := tx.Table("ai_rate_cards").Create(row).Error; err != nil {
			return err
		}
		var id int64
		if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&id).Error; err != nil {
			return err
		}
		if publish {
			if err := tx.Table("ai_rate_cards").Where("provider_key = ? AND model_key = ? AND id <> ? AND status = 'published'", providerKey, modelKey, id).Updates(map[string]any{"status": "retired", "retired_at": now}).Error; err != nil {
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
		Start, End                            time.Time
		Cancel                                bool
	}
	var value row
	err := s.db.WithContext(ctx).Raw(`SELECT subscription.id, product.product_key, product.name product_name, subscription.status, subscription.term,
		subscription.current_period_start start, subscription.current_period_end end, subscription.cancel_at_period_end cancel
		FROM billing_subscriptions subscription JOIN billing_products product ON product.id = subscription.product_id
		WHERE subscription.owner_type = ? AND subscription.owner_id = ? AND subscription.status = 'active'
		ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, owner.Type, owner.ID).Scan(&value).Error
	if err != nil {
		return nil, err
	}
	if value.ID == 0 {
		return nil, nil
	}
	return &pb.BillingSubscriptionInfo{Id: value.ID, ProductKey: value.ProductKey, ProductName: value.ProductName, Status: value.Status, Term: value.Term, CurrentPeriodStartUnixMs: value.Start.UnixMilli(), CurrentPeriodEndUnixMs: value.End.UnixMilli(), CancelAtPeriodEnd: value.Cancel}, nil
}

func (s *Commerce) ListOrders(ctx context.Context, owner model.Owner, page, pageSize int32) ([]*pb.BillingOrderInfo, int64, error) {
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

func (s *Commerce) CreateOrder(ctx context.Context, owner model.Owner, actorUserID, priceVersionID uint64, orderType, idempotencyKey string) (*pb.BillingOrderInfo, error) {
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	if actorUserID == 0 || priceVersionID == 0 || strings.TrimSpace(idempotencyKey) == "" {
		return nil, errors.New("actor, price and idempotency key are required")
	}
	if orderType != "subscribe" && orderType != "renew" && orderType != "upgrade" && orderType != "credit_pack" {
		return nil, errors.New("invalid billing order type")
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
	now := s.now().UTC()
	if price.ProductType == "subscription" {
		var openOrderCount int64
		if err := s.db.WithContext(ctx).Table("billing_orders").Where("owner_type = ? AND owner_id = ? AND order_type IN ('subscribe','renew','upgrade') AND status IN ('pending','paying') AND expires_at > ?", owner.Type, owner.ID, now).Count(&openOrderCount).Error; err != nil {
			return nil, err
		}
		if openOrderCount > 0 {
			return nil, errors.New("an unpaid subscription order already exists")
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
				return nil, errors.New("a renewal is already scheduled")
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
	orderNo := "B" + now.Format("20060102150405") + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:12])
	row := map[string]any{"order_no": orderNo, "owner_type": owner.Type, "owner_id": owner.ID, "order_type": orderType, "product_id": price.ProductID, "price_version_id": priceVersionID, "amount_fen": amountFen, "currency": price.Currency, "status": "pending", "payment_environment": s.Environment(), "idempotency_key": idempotencyKey, "expires_at": now.Add(30 * time.Minute), "created_at": now, "updated_at": now}
	err := s.db.WithContext(ctx).Table("billing_orders").Create(row).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return s.getOrderByIdempotency(ctx, owner, idempotencyKey)
	}
	if err != nil {
		return nil, err
	}
	return orderProto(orderNo, orderType, price.ProductKey, price.ProductName, price.Currency, "pending", s.Environment(), int64(priceVersionID), int64(amountFen), now.Add(30*time.Minute), nil, now), nil
}

func (s *Commerce) CreatePayment(ctx context.Context, owner model.Owner, orderNo, scene string) (string, string, error) {
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
		return "", "", err
	}
	if order.ID == 0 || order.Status != "pending" || !order.ExpiresAt.After(s.now().UTC()) || order.Environment != s.Environment() {
		return "", "", errors.New("billing order cannot be paid")
	}
	paymentNo := "P" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:24]
	redirectURL, err := s.alipay.PayURL(payment.PayRequest{OrderNo: order.OrderNo, Subject: order.ProductName, Scene: scene, AmountFen: order.AmountFen}, s.now())
	if err != nil {
		return "", "", err
	}
	create := map[string]any{"order_id": order.ID, "payment_no": paymentNo, "channel": "alipay", "scene": scene, "payment_environment": s.Environment(), "amount_fen": order.AmountFen, "currency": "CNY", "status": "pending", "pay_payload": redirectURL, "created_at": s.now().UTC(), "updated_at": s.now().UTC()}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("billing_payments").Create(create).Error; err != nil {
			return err
		}
		return tx.Table("billing_orders").Where("id = ? AND status = 'pending'", order.ID).Update("status", "paying").Error
	}); err != nil {
		return "", "", err
	}
	return paymentNo, redirectURL, nil
}

func (s *Commerce) ProcessAlipayNotification(ctx context.Context, fields map[string]string) error {
	if err := s.alipay.VerifyNotification(fields); err != nil {
		return err
	}
	if fields["trade_status"] != "TRADE_SUCCESS" && fields["trade_status"] != "TRADE_FINISHED" {
		return nil
	}
	payload, _ := json.Marshal(fields)
	digest := sha256.Sum256(payload)
	eventKey := fields["notify_id"]
	if eventKey == "" {
		eventKey = fields["trade_no"] + ":" + fields["trade_status"]
	}
	now := s.now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		event := map[string]any{"channel": "alipay", "payment_environment": s.Environment(), "event_key": eventKey, "event_type": fields["notify_type"], "signature_verified": true, "payload_sha256": hex.EncodeToString(digest[:]), "payload": string(payload), "status": "verified", "created_at": now}
		if err := tx.Table("billing_webhook_events").Create(event).Error; errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		} else if err != nil {
			return err
		}
		var order commerceOrderRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("billing_orders").Where("order_no = ?", fields["out_trade_no"]).Scan(&order).Error; err != nil {
			return err
		}
		amountFen, err := parseAmountFen(fields["total_amount"])
		if order.ID == 0 || err != nil || amountFen != order.AmountFen || order.Currency != "CNY" || order.Environment != s.Environment() {
			return errors.New("Alipay notification order, amount, currency or environment mismatch")
		}
		if order.Status == "paid" {
			return tx.Table("billing_webhook_events").Where("event_key = ? AND payment_environment = ?", eventKey, s.Environment()).Updates(map[string]any{"status": "processed", "processed_at": now}).Error
		}
		if order.Status != "paying" && order.Status != "pending" {
			return errors.New("billing order is not payable")
		}
		if err := tx.Table("billing_payments").Where("order_id = ? AND payment_environment = ?", order.ID, s.Environment()).Updates(map[string]any{"status": "succeeded", "channel_trade_no": fields["trade_no"], "paid_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Table("billing_orders").Where("id = ?", order.ID).Updates(map[string]any{"status": "paid", "paid_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := s.activateOrder(ctx, tx, order, now); err != nil {
			return err
		}
		return tx.Table("billing_webhook_events").Where("event_key = ? AND payment_environment = ?", eventKey, s.Environment()).Updates(map[string]any{"status": "processed", "processed_at": now}).Error
	})
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

func (s *Commerce) RequestRefund(ctx context.Context, owner model.Owner, actorUserID uint64, orderNo, reason string) (string, string, string, error) {
	if actorUserID == 0 || strings.TrimSpace(reason) == "" {
		return "", "", "", errors.New("refund actor and reason are required")
	}
	type row struct {
		ID, PaymentID, AmountFen uint64
		OrderType, Status        string
		PaidAt                   time.Time
	}
	var order row
	if err := s.db.WithContext(ctx).Raw(`SELECT orders.id, orders.order_type, orders.status, orders.amount_fen, orders.paid_at, payment.id payment_id
		FROM billing_orders orders JOIN billing_payments payment ON payment.order_id = orders.id AND payment.status = 'succeeded'
		WHERE orders.order_no = ? AND orders.owner_type = ? AND orders.owner_id = ?`, orderNo, owner.Type, owner.ID).Scan(&order).Error; err != nil {
		return "", "", "", err
	}
	if order.ID == 0 || order.Status != "paid" {
		return "", "", "", errors.New("only paid orders can be refunded")
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
	if err := s.db.WithContext(ctx).Table("billing_refunds").Create(map[string]any{"refund_no": refundNo, "order_id": order.ID, "payment_id": order.PaymentID, "amount_fen": order.AmountFen, "reason": reason, "status": statusValue, "review_mode": reviewMode, "requested_by": actorUserID, "created_at": now, "updated_at": now}).Error; err != nil {
		return "", "", "", err
	}
	if reviewMode == "manual" {
		return refundNo, statusValue, reviewMode, nil
	}
	channelNo, err := s.alipay.Refund(ctx, orderNo, refundNo, reason, order.AmountFen, now)
	if err != nil {
		_ = s.db.WithContext(ctx).Table("billing_refunds").Where("refund_no = ?", refundNo).Updates(map[string]any{"status": "failed", "updated_at": now}).Error
		return refundNo, "failed", reviewMode, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("billing_refunds").Where("refund_no = ?", refundNo).Updates(map[string]any{"status": "succeeded", "channel_refund_no": channelNo, "succeeded_at": now, "updated_at": now}).Error; err != nil {
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
	orders, _, err := s.ListOrders(ctx, owner, 1, 100)
	if err != nil {
		return nil, err
	}
	var orderNo string
	if err := s.db.WithContext(ctx).Table("billing_orders").Where("owner_type = ? AND owner_id = ? AND idempotency_key = ?", owner.Type, owner.ID, key).Select("order_no").Scan(&orderNo).Error; err != nil {
		return nil, err
	}
	for _, order := range orders {
		if order.OrderNo == orderNo {
			return order, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
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
