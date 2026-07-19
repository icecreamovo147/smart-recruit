package payment

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/url"
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
	payURL, err := client.PayURL(PayRequest{OrderNo: "B20260720001", Subject: "AI Pro", Scene: "desktop", AmountFen: 9900}, time.Unix(1000, 0))
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(payURL)
	if parsed.Host != "openapi-sandbox.dl.alipaydev.com" || parsed.Query().Get("method") != "alipay.trade.page.pay" || parsed.Query().Get("sign") == "" {
		t.Fatalf("pay URL = %s", payURL)
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
