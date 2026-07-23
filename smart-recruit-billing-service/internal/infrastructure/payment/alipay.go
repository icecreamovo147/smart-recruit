package payment

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultSandboxGateway = "https://openapi-sandbox.dl.alipaydev.com/gateway.do"

const alipayStatusRequestTimeout = 4 * time.Second

type AlipayConfig struct {
	Environment     string
	GatewayURL      string
	AppID           string
	PrivateKey      string
	VerifyPublicKey string
	SellerID        string
	NotifyURL       string
	ReturnURL       string
	DesktopEnabled  bool
	WAPEnabled      bool
}

func (c AlipayConfig) Validate() error {
	if c.Environment != "sandbox" {
		return errors.New("development billing only permits ALIPAY_ENV=sandbox")
	}
	if c.GatewayURL == "" {
		c.GatewayURL = DefaultSandboxGateway
	}
	parsed, err := url.Parse(c.GatewayURL)
	if err != nil || parsed.Scheme != "https" || !strings.HasSuffix(parsed.Hostname(), "alipaydev.com") {
		return errors.New("sandbox Alipay gateway must be an HTTPS alipaydev.com endpoint")
	}
	if strings.TrimSpace(c.AppID) == "" || strings.TrimSpace(c.PrivateKey) == "" || strings.TrimSpace(c.VerifyPublicKey) == "" || strings.TrimSpace(c.SellerID) == "" {
		return errors.New("Alipay sandbox app, seller and RSA2 keys are required")
	}
	if strings.TrimSpace(c.NotifyURL) == "" {
		return errors.New("Alipay notify URL is required")
	}
	notifyURL, err := url.Parse(c.NotifyURL)
	if err != nil || notifyURL.Scheme != "https" || notifyURL.Host == "" || notifyURL.User != nil {
		return errors.New("Alipay notify URL must be an absolute HTTPS URL")
	}
	returnURL, err := url.Parse(c.ReturnURL)
	if err != nil || (returnURL.Scheme != "http" && returnURL.Scheme != "https") || returnURL.Host == "" || returnURL.User != nil {
		return errors.New("Alipay return URL must be an absolute HTTP(S) URL")
	}
	return nil
}

type Alipay struct {
	config           AlipayConfig
	privateKey       *rsa.PrivateKey
	publicKey        *rsa.PublicKey
	httpClient       *http.Client
	breakerMu        sync.Mutex
	breakerFailures  int
	breakerOpenUntil time.Time
}

type UnavailableAlipay struct{ reason error }

func NewUnavailableAlipay(reason error) *UnavailableAlipay {
	if reason == nil {
		reason = errors.New("Alipay sandbox is not configured")
	}
	return &UnavailableAlipay{reason: reason}
}
func (*UnavailableAlipay) Environment() string                            { return "sandbox" }
func (a *UnavailableAlipay) PayURL(PayRequest, time.Time) (string, error) { return "", a.reason }
func (a *UnavailableAlipay) VerifyNotification(map[string]string) error   { return a.reason }
func (a *UnavailableAlipay) VerifyReturn(map[string]string) error         { return a.reason }
func (a *UnavailableAlipay) Refund(context.Context, string, string, string, uint64, time.Time) (string, error) {
	return "", a.reason
}
func (a *UnavailableAlipay) QueryRefund(context.Context, string, string, time.Time) (RefundQueryResult, error) {
	return RefundQueryResult{}, a.reason
}
func (a *UnavailableAlipay) Query(context.Context, string, time.Time) (QueryResult, error) {
	return QueryResult{}, a.reason
}
func (a *UnavailableAlipay) Close(context.Context, string, time.Time) error { return a.reason }

func NewAlipay(config AlipayConfig) (*Alipay, error) {
	if config.GatewayURL == "" {
		config.GatewayURL = DefaultSandboxGateway
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	privateKey, err := parsePrivateKey(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse Alipay application private key: %w", err)
	}
	publicKey, err := parsePublicKey(config.VerifyPublicKey)
	if err != nil {
		return nil, fmt.Errorf("parse Alipay public key: %w", err)
	}
	if privateKey.PublicKey.E == publicKey.E && privateKey.PublicKey.N.Cmp(publicKey.N) == 0 {
		return nil, errors.New("Alipay verify public key is the application public key; configure the Alipay public key")
	}
	if privateKey.N.BitLen() < 2048 || publicKey.N.BitLen() < 2048 {
		return nil, errors.New("Alipay RSA2 keys must be at least 2048 bits")
	}
	if err := privateKey.Validate(); err != nil {
		return nil, fmt.Errorf("validate Alipay application private key: %w", err)
	}
	return &Alipay{
		config:     config,
		privateKey: privateKey,
		publicKey:  publicKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			// OpenAPI query/close/refund calls must return protocol JSON. Following
			// a sandbox redirect hides the original response behind an HTML error
			// page and can consume the gateway's entire request deadline.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (a *Alipay) Environment() string { return a.config.Environment }

type PayRequest struct {
	OrderNo, Subject, Scene string
	ReturnToken             string
	AmountFen               uint64
	ExpiresAt               time.Time
}

func (a *Alipay) PayURL(request PayRequest, now time.Time) (string, error) {
	if request.OrderNo == "" || request.Subject == "" || request.AmountFen == 0 || request.ExpiresAt.IsZero() {
		return "", errors.New("Alipay order number, subject, positive amount and expiry are required")
	}
	if !request.ExpiresAt.After(now) {
		return "", errors.New("Alipay order has expired")
	}
	method := "alipay.trade.page.pay"
	productCode := "FAST_INSTANT_TRADE_PAY"
	if request.Scene == "wap" {
		method = "alipay.trade.wap.pay"
		productCode = "QUICK_WAP_WAY"
		if !a.config.WAPEnabled {
			return "", errors.New("Alipay WAP payment is disabled")
		}
	} else if request.Scene == "desktop" {
		if !a.config.DesktopEnabled {
			return "", errors.New("Alipay desktop payment is disabled")
		}
	} else {
		return "", errors.New("Alipay scene must be desktop or wap")
	}
	location, _ := time.LoadLocation("Asia/Shanghai")
	biz, err := json.Marshal(map[string]any{
		"out_trade_no": request.OrderNo,
		"total_amount": formatFen(request.AmountFen),
		"subject":      request.Subject,
		"product_code": productCode,
		"time_expire":  request.ExpiresAt.In(location).Format("2006-01-02 15:04:05"),
	})
	if err != nil {
		return "", err
	}
	params := map[string]string{
		"app_id": a.config.AppID, "method": method, "format": "JSON", "charset": "utf-8",
		"sign_type": "RSA2", "timestamp": now.In(location).Format("2006-01-02 15:04:05"), "version": "1.0",
		"notify_url": a.config.NotifyURL, "biz_content": string(biz),
	}
	if a.config.ReturnURL != "" {
		returnURL := a.config.ReturnURL
		if request.ReturnToken != "" {
			parsed, err := url.Parse(returnURL)
			if err != nil {
				return "", errors.New("Alipay return URL is invalid")
			}
			query := parsed.Query()
			query.Set("return_token", request.ReturnToken)
			parsed.RawQuery = query.Encode()
			returnURL = parsed.String()
		}
		params["return_url"] = returnURL
	}
	signature, err := a.sign(canonical(params))
	if err != nil {
		return "", err
	}
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	values.Set("sign", signature)
	return a.config.GatewayURL + "?" + values.Encode(), nil
}

func (a *Alipay) VerifyNotification(fields map[string]string) error {
	if fields["app_id"] != a.config.AppID {
		return errors.New("Alipay notification app_id mismatch")
	}
	if fields["seller_id"] != a.config.SellerID {
		return errors.New("Alipay notification seller_id mismatch")
	}
	signature := fields["sign"]
	if signature == "" || fields["sign_type"] != "RSA2" {
		return errors.New("Alipay RSA2 signature is required")
	}
	copyFields := make(map[string]string, len(fields))
	for key, value := range fields {
		if key != "sign" && key != "sign_type" && value != "" {
			copyFields[key] = value
		}
	}
	digest := sha256.Sum256([]byte(canonical(copyFields)))
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return errors.New("Alipay signature is not valid base64")
	}
	if err := rsa.VerifyPKCS1v15(a.publicKey, crypto.SHA256, digest[:], decoded); err != nil {
		return errors.New("Alipay notification signature verification failed")
	}
	return nil
}

func (a *Alipay) VerifyReturn(fields map[string]string) error {
	if fields["app_id"] != a.config.AppID {
		return errors.New("Alipay return app_id mismatch")
	}
	signature := fields["sign"]
	if signature == "" || fields["sign_type"] != "RSA2" {
		return errors.New("Alipay return RSA2 signature is required")
	}
	copyFields := make(map[string]string, len(fields))
	for key, value := range fields {
		if key != "sign" && key != "sign_type" && key != "return_token" && value != "" {
			copyFields[key] = value
		}
	}
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return errors.New("Alipay return signature is not valid base64")
	}
	digest := sha256.Sum256([]byte(canonical(copyFields)))
	if err := rsa.VerifyPKCS1v15(a.publicKey, crypto.SHA256, digest[:], decoded); err != nil {
		return errors.New("Alipay return signature verification failed")
	}
	return nil
}

func (a *Alipay) Refund(ctx context.Context, orderNo, refundNo, reason string, amountFen uint64, now time.Time) (string, error) {
	if orderNo == "" || refundNo == "" || amountFen == 0 {
		return "", errors.New("Alipay refund identifiers and amount are required")
	}
	biz, err := json.Marshal(map[string]any{"out_trade_no": orderNo, "refund_amount": formatFen(amountFen), "refund_reason": reason, "out_request_no": refundNo})
	if err != nil {
		return "", err
	}
	location, _ := time.LoadLocation("Asia/Shanghai")
	params := map[string]string{"app_id": a.config.AppID, "method": "alipay.trade.refund", "format": "JSON", "charset": "utf-8", "sign_type": "RSA2", "timestamp": now.In(location).Format("2006-01-02 15:04:05"), "version": "1.0", "biz_content": string(biz)}
	signature, err := a.sign(canonical(params))
	if err != nil {
		return "", err
	}
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	values.Set("sign", signature)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.config.GatewayURL, strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	response, err := a.do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if err := a.verifyAPIResponseHTTP("refund", response, body, "alipay_trade_refund_response"); err != nil {
		return "", err
	}
	var envelope struct {
		Response struct {
			Code       string `json:"code"`
			Msg        string `json:"msg"`
			SubMsg     string `json:"sub_msg"`
			TradeNo    string `json:"trade_no"`
			OutTradeNo string `json:"out_trade_no"`
			RefundFee  string `json:"refund_fee"`
		} `json:"alipay_trade_refund_response"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", fmt.Errorf("decode Alipay refund response: %w", err)
	}
	if envelope.Response.Code != "10000" {
		return "", fmt.Errorf("Alipay refund rejected: %s %s", envelope.Response.Msg, envelope.Response.SubMsg)
	}
	if envelope.Response.OutTradeNo != orderNo {
		return "", errors.New("Alipay refund response order mismatch")
	}
	refundAmount, err := parseFen(envelope.Response.RefundFee)
	if err != nil || refundAmount != amountFen {
		return "", errors.New("Alipay refund response amount mismatch")
	}
	return envelope.Response.TradeNo, nil
}

type QueryResult struct {
	MerchantOrderNo, TradeNo, TradeStatus string
	AmountFen                             uint64
}

type RefundQueryResult struct {
	TradeNo, RefundNo, RefundStatus string
	AmountFen                       uint64
}

type APIError struct {
	Operation, Code, SubCode, Message, SubMessage string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Alipay %s rejected: %s %s (%s)", e.Operation, e.Message, e.SubMessage, e.SubCode)
}

func IsTradeNotExist(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && (apiErr.SubCode == "ACQ.TRADE_NOT_EXIST" || strings.Contains(apiErr.SubMessage, "交易不存在"))
}

func IsAPIRejected(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr)
}

func (a *Alipay) Query(ctx context.Context, orderNo string, now time.Time) (QueryResult, error) {
	if strings.TrimSpace(orderNo) == "" {
		return QueryResult{}, errors.New("Alipay order number is required")
	}
	biz, _ := json.Marshal(map[string]string{"out_trade_no": orderNo})
	location, _ := time.LoadLocation("Asia/Shanghai")
	params := map[string]string{"app_id": a.config.AppID, "method": "alipay.trade.query", "format": "JSON", "charset": "utf-8", "sign_type": "RSA2", "timestamp": now.In(location).Format("2006-01-02 15:04:05"), "version": "1.0", "biz_content": string(biz)}
	signature, err := a.sign(canonical(params))
	if err != nil {
		return QueryResult{}, err
	}
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	values.Set("sign", signature)
	requestCtx, cancel := boundedStatusContext(ctx)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, a.config.GatewayURL, strings.NewReader(values.Encode()))
	if err != nil {
		return QueryResult{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	response, err := a.do(request)
	if err != nil {
		return QueryResult{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return QueryResult{}, err
	}
	if err := a.verifyAPIResponseHTTP("query", response, body, "alipay_trade_query_response"); err != nil {
		return QueryResult{}, err
	}
	var envelope struct {
		Response struct {
			Code        string `json:"code"`
			Msg         string `json:"msg"`
			SubCode     string `json:"sub_code"`
			SubMsg      string `json:"sub_msg"`
			TradeNo     string `json:"trade_no"`
			OutTradeNo  string `json:"out_trade_no"`
			TradeStatus string `json:"trade_status"`
			TotalAmount string `json:"total_amount"`
		} `json:"alipay_trade_query_response"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return QueryResult{}, nonJSONResponseError("query", response, err)
	}
	if envelope.Response.Code != "10000" {
		return QueryResult{}, &APIError{Operation: "query", Code: envelope.Response.Code, SubCode: envelope.Response.SubCode, Message: envelope.Response.Msg, SubMessage: envelope.Response.SubMsg}
	}
	if envelope.Response.OutTradeNo != orderNo {
		return QueryResult{}, errors.New("Alipay query response order mismatch")
	}
	var amount uint64
	if envelope.Response.TotalAmount != "" {
		amount, err = parseFen(envelope.Response.TotalAmount)
		if err != nil {
			return QueryResult{}, err
		}
	} else if envelope.Response.TradeStatus == "TRADE_SUCCESS" || envelope.Response.TradeStatus == "TRADE_FINISHED" {
		return QueryResult{}, errors.New("Alipay successful trade query omitted total amount")
	}
	return QueryResult{MerchantOrderNo: envelope.Response.OutTradeNo, TradeNo: envelope.Response.TradeNo, TradeStatus: envelope.Response.TradeStatus, AmountFen: amount}, nil
}

func (a *Alipay) Close(ctx context.Context, merchantOrderNo string, now time.Time) error {
	if strings.TrimSpace(merchantOrderNo) == "" {
		return errors.New("Alipay merchant order number is required")
	}
	biz, _ := json.Marshal(map[string]string{"out_trade_no": merchantOrderNo})
	location, _ := time.LoadLocation("Asia/Shanghai")
	params := map[string]string{"app_id": a.config.AppID, "method": "alipay.trade.close", "format": "JSON", "charset": "utf-8", "sign_type": "RSA2", "timestamp": now.In(location).Format("2006-01-02 15:04:05"), "version": "1.0", "biz_content": string(biz)}
	signature, err := a.sign(canonical(params))
	if err != nil {
		return err
	}
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	values.Set("sign", signature)
	requestCtx, cancel := boundedStatusContext(ctx)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, a.config.GatewayURL, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	response, err := a.do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if err := a.verifyAPIResponseHTTP("close", response, body, "alipay_trade_close_response"); err != nil {
		return err
	}
	var envelope struct {
		Response struct {
			Code    string `json:"code"`
			Msg     string `json:"msg"`
			SubCode string `json:"sub_code"`
			SubMsg  string `json:"sub_msg"`
		} `json:"alipay_trade_close_response"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nonJSONResponseError("close", response, err)
	}
	if envelope.Response.Code == "10000" || envelope.Response.SubCode == "ACQ.TRADE_NOT_EXIST" {
		return nil
	}
	return &APIError{Operation: "close", Code: envelope.Response.Code, SubCode: envelope.Response.SubCode, Message: envelope.Response.Msg, SubMessage: envelope.Response.SubMsg}
}

func (a *Alipay) QueryRefund(ctx context.Context, merchantOrderNo, refundNo string, now time.Time) (RefundQueryResult, error) {
	if strings.TrimSpace(merchantOrderNo) == "" || strings.TrimSpace(refundNo) == "" {
		return RefundQueryResult{}, errors.New("Alipay refund query identifiers are required")
	}
	biz, _ := json.Marshal(map[string]string{"out_trade_no": merchantOrderNo, "out_request_no": refundNo})
	location, _ := time.LoadLocation("Asia/Shanghai")
	params := map[string]string{"app_id": a.config.AppID, "method": "alipay.trade.fastpay.refund.query", "format": "JSON", "charset": "utf-8", "sign_type": "RSA2", "timestamp": now.In(location).Format("2006-01-02 15:04:05"), "version": "1.0", "biz_content": string(biz)}
	signature, err := a.sign(canonical(params))
	if err != nil {
		return RefundQueryResult{}, err
	}
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	values.Set("sign", signature)
	requestCtx, cancel := boundedStatusContext(ctx)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, a.config.GatewayURL, strings.NewReader(values.Encode()))
	if err != nil {
		return RefundQueryResult{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	response, err := a.do(request)
	if err != nil {
		return RefundQueryResult{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return RefundQueryResult{}, err
	}
	if err := a.verifyAPIResponseHTTP("refund query", response, body, "alipay_trade_fastpay_refund_query_response"); err != nil {
		return RefundQueryResult{}, err
	}
	var envelope struct {
		Response struct {
			Code         string `json:"code"`
			Msg          string `json:"msg"`
			SubCode      string `json:"sub_code"`
			SubMsg       string `json:"sub_msg"`
			TradeNo      string `json:"trade_no"`
			RefundNo     string `json:"out_request_no"`
			RefundStatus string `json:"refund_status"`
			RefundAmount string `json:"refund_amount"`
		} `json:"alipay_trade_fastpay_refund_query_response"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return RefundQueryResult{}, nonJSONResponseError("refund query", response, err)
	}
	if envelope.Response.Code != "10000" {
		return RefundQueryResult{}, &APIError{Operation: "refund query", Code: envelope.Response.Code, SubCode: envelope.Response.SubCode, Message: envelope.Response.Msg, SubMessage: envelope.Response.SubMsg}
	}
	amount, err := parseFen(envelope.Response.RefundAmount)
	if err != nil {
		return RefundQueryResult{}, fmt.Errorf("parse Alipay refund amount: %w", err)
	}
	return RefundQueryResult{TradeNo: envelope.Response.TradeNo, RefundNo: envelope.Response.RefundNo, RefundStatus: envelope.Response.RefundStatus, AmountFen: amount}, nil
}

func (a *Alipay) verifyAPIResponse(body []byte, responseField string) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode Alipay response envelope: %w", err)
	}
	raw, ok := envelope[responseField]
	if !ok || len(raw) == 0 {
		return fmt.Errorf("Alipay response omitted %s", responseField)
	}
	var signature string
	if value, ok := envelope["sign"]; ok {
		_ = json.Unmarshal(value, &signature)
	}
	if signature == "" {
		return errors.New("Alipay API response signature is required")
	}
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return errors.New("Alipay API response signature is not valid base64")
	}
	digest := sha256.Sum256(raw)
	if err := rsa.VerifyPKCS1v15(a.publicKey, crypto.SHA256, digest[:], decoded); err != nil {
		return errors.New("Alipay API response signature verification failed")
	}
	return nil
}

func (a *Alipay) verifyAPIResponseHTTP(operation string, response *http.Response, body []byte, responseField string) error {
	err := a.verifyAPIResponse(body, responseField)
	if err != nil && !json.Valid(body) {
		return nonJSONResponseError(operation, response, err)
	}
	return err
}

func (a *Alipay) do(request *http.Request) (*http.Response, error) {
	now := time.Now()
	a.breakerMu.Lock()
	if now.Before(a.breakerOpenUntil) {
		a.breakerMu.Unlock()
		return nil, errors.New("Alipay API circuit is open after repeated gateway failures")
	}
	a.breakerMu.Unlock()
	response, err := a.httpClient.Do(request)
	a.breakerMu.Lock()
	defer a.breakerMu.Unlock()
	if err != nil || (response != nil && response.StatusCode >= http.StatusInternalServerError) {
		a.breakerFailures++
		if a.breakerFailures >= 5 {
			a.breakerOpenUntil = now.Add(30 * time.Second)
			a.breakerFailures = 0
		}
	} else {
		a.breakerFailures = 0
		a.breakerOpenUntil = time.Time{}
	}
	return response, err
}

func boundedStatusContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= alipayStatusRequestTimeout {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, alipayStatusRequestTimeout)
}

func nonJSONResponseError(operation string, response *http.Response, decodeErr error) error {
	contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
	return fmt.Errorf("Alipay %s returned non-JSON response (HTTP %d, content-type %q): %w", operation, response.StatusCode, contentType, decodeErr)
}

func (a *Alipay) sign(content string) (string, error) {
	digest := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, a.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func canonical(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if value != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func formatFen(value uint64) string { return fmt.Sprintf("%d.%02d", value/100, value%100) }

func parseFen(value string) (uint64, error) {
	parts := strings.Split(value, ".")
	if len(parts) == 0 || len(parts) > 2 {
		return 0, errors.New("invalid Alipay amount")
	}
	yuan, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	fraction := "00"
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
		if len(fraction) != 2 {
			return 0, errors.New("invalid Alipay amount precision")
		}
	}
	fen, err := strconv.ParseUint(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	return yuan*100 + fen, nil
}

func parsePrivateKey(value string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(value, "PRIVATE KEY")))
	if block == nil {
		return nil, errors.New("private key PEM is invalid")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func parsePublicKey(value string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(value, "PUBLIC KEY")))
	if block == nil {
		return nil, errors.New("public key PEM is invalid")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		if rsaKey, pkcs1Err := x509.ParsePKCS1PublicKey(block.Bytes); pkcs1Err == nil {
			return rsaKey, nil
		}
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}
	return rsaKey, nil
}

func normalizePEM(value, kind string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, `\n`, "\n"))
	if strings.Contains(trimmed, "-----BEGIN") {
		return trimmed
	}
	return "-----BEGIN " + kind + "-----\n" + trimmed + "\n-----END " + kind + "-----"
}
