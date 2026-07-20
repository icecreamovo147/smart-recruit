package payment

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSandboxPayURLAndNotificationVerification(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustPKCS8(t, key)})
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: mustPublic(t, &key.PublicKey)})
	client, err := NewAlipay(AlipayConfig{Environment: "sandbox", AppID: "sandbox-app", SellerID: "seller", PrivateKey: string(privatePEM), PublicKey: string(publicPEM), NotifyURL: "https://example.test/notify", ReturnURL: "https://example.test/return", DesktopEnabled: true, WAPEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.July, 20, 2, 0, 0, 0, time.UTC)
	expiresAt := now.Add(30 * time.Minute)
	payURL, err := client.PayURL(PayRequest{OrderNo: "B20260720001", Subject: "AI Pro", Scene: "desktop", AmountFen: 9900, ExpiresAt: expiresAt}, now)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(payURL)
	if parsed.Host != "openapi-sandbox.dl.alipaydev.com" || parsed.Query().Get("method") != "alipay.trade.page.pay" || parsed.Query().Get("sign") == "" {
		t.Fatalf("pay URL = %s", payURL)
	}
	var bizContent map[string]any
	if err := json.Unmarshal([]byte(parsed.Query().Get("biz_content")), &bizContent); err != nil {
		t.Fatalf("decode biz_content: %v", err)
	}
	if bizContent["time_expire"] != "2026-07-20 10:30:00" {
		t.Fatalf("time_expire = %#v", bizContent["time_expire"])
	}
	if _, exists := bizContent["timeout_express"]; exists {
		t.Fatal("payment timeout must use the order's fixed absolute expiry")
	}
	fields := map[string]string{"app_id": "sandbox-app", "seller_id": "seller", "out_trade_no": "B20260720001", "trade_status": "TRADE_SUCCESS", "total_amount": "99.00", "sign_type": "RSA2"}
	signature, err := client.sign(canonical(fieldsWithoutSignType(fields)))
	if err != nil {
		t.Fatal(err)
	}
	fields["sign"] = signature
	if err := client.VerifyNotification(fields); err != nil {
		t.Fatal(err)
	}
	fields["total_amount"] = "1.00"
	if err := client.VerifyNotification(fields); err == nil {
		t.Fatal("expected tampered notification to fail")
	}
}

func TestQueryAndCloseParseSnakeCaseResponses(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustPKCS8(t, key)})
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: mustPublic(t, &key.PublicKey)})
	client, err := NewAlipay(AlipayConfig{Environment: "sandbox", AppID: "sandbox-app", SellerID: "seller", PrivateKey: string(privatePEM), PublicKey: string(publicPEM), NotifyURL: "https://example.test/notify", DesktopEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Form.Get("method") {
		case "alipay.trade.query":
			_, _ = w.Write([]byte(`{"alipay_trade_query_response":{"code":"10000","msg":"Success","trade_no":"20260720001","trade_status":"TRADE_SUCCESS","total_amount":"39.00"}}`))
		case "alipay.trade.close":
			_, _ = w.Write([]byte(`{"alipay_trade_close_response":{"code":"40004","msg":"Business Failed","sub_code":"ACQ.TRADE_NOT_EXIST","sub_msg":"trade does not exist"}}`))
		default:
			http.Error(w, "unexpected method", http.StatusBadRequest)
		}
	}))
	defer server.Close()
	client.config.GatewayURL = server.URL
	client.httpClient = server.Client()

	result, err := client.Query(context.Background(), "P20260720001", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if result.TradeNo != "20260720001" || result.TradeStatus != "TRADE_SUCCESS" || result.AmountFen != 3900 {
		t.Fatalf("query result = %#v", result)
	}
	if err := client.Close(context.Background(), "P20260720002", time.Now()); err != nil {
		t.Fatalf("close missing trade: %v", err)
	}
}

func TestPayURLRejectsExpiredOrder(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustPKCS8(t, key)})
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: mustPublic(t, &key.PublicKey)})
	client, err := NewAlipay(AlipayConfig{Environment: "sandbox", AppID: "sandbox-app", SellerID: "seller", PrivateKey: string(privatePEM), PublicKey: string(publicPEM), NotifyURL: "https://example.test/notify", DesktopEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.July, 20, 2, 0, 0, 0, time.UTC)
	_, err = client.PayURL(PayRequest{OrderNo: "P20260720003", Subject: "AI Basic", Scene: "desktop", AmountFen: 3900, ExpiresAt: now}, now)
	if err == nil {
		t.Fatal("expected an expired order to be rejected")
	}
}

func TestIsTradeNotExist(t *testing.T) {
	err := &APIError{Operation: "query", Code: "40004", SubCode: "ACQ.TRADE_NOT_EXIST", Message: "Business Failed", SubMessage: "交易不存在"}
	if !IsTradeNotExist(err) {
		t.Fatal("expected ACQ.TRADE_NOT_EXIST to be recognized")
	}
	if IsTradeNotExist(errors.New("network unavailable")) {
		t.Fatal("unexpected missing-trade classification")
	}
}

func TestStatusAPIClientDoesNotFollowHTMLRedirect(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustPKCS8(t, key)})
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: mustPublic(t, &key.PublicKey)})
	client, err := NewAlipay(AlipayConfig{Environment: "sandbox", AppID: "sandbox-app", SellerID: "seller", PrivateKey: string(privatePEM), PublicKey: string(publicPEM), NotifyURL: "https://example.test/notify", DesktopEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	redirects := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/error" {
			redirects++
			_, _ = w.Write([]byte("<html>error</html>"))
			return
		}
		http.Redirect(w, r, "/error", http.StatusFound)
	}))
	defer server.Close()
	client.config.GatewayURL = server.URL

	err = client.Close(context.Background(), "P20260720004", time.Now())
	if err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("close error = %v, want original redirect response", err)
	}
	if redirects != 0 {
		t.Fatalf("followed %d sandbox error redirects", redirects)
	}
}

func fieldsWithoutSignType(fields map[string]string) map[string]string {
	result := map[string]string{}
	for key, value := range fields {
		if key != "sign_type" {
			result[key] = value
		}
	}
	return result
}

func mustPKCS8(t *testing.T, key *rsa.PrivateKey) []byte {
	value, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
func mustPublic(t *testing.T, key *rsa.PublicKey) []byte {
	value, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
