package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	platformserver "smart-recruit-platform-go/server"
	"smart-recruit-proto/recruitment/pb"

	"smart-recruit-billing-service/internal/application/service"
	billingconfig "smart-recruit-billing-service/internal/config"
	"smart-recruit-billing-service/internal/domain/model"
	"smart-recruit-billing-service/internal/infrastructure/payment"
	"smart-recruit-billing-service/internal/infrastructure/persistence"
	billinggrpc "smart-recruit-billing-service/internal/interfaces/grpc"
	billingruntime "smart-recruit-billing-service/internal/runtime"
)

func main() {
	check := flag.Bool("check", false, "validate Billing service runtime wiring and exit")
	serve := flag.Bool("serve", false, "start Billing gRPC runtime")
	addr := flag.String("addr", envOrDefault("GRPC_ADDR", ":50069"), "Billing gRPC listen address")
	configPath := flag.String("config", "", "Billing YAML config path; when omitted, legacy environment variables are used")
	flag.Parse()
	if *check {
		if err := checkRuntime(); err != nil {
			fmt.Fprintf(os.Stderr, "billing-service check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "billing-service runtime check passed")
		return
	}
	if !*serve {
		fmt.Fprintln(os.Stderr, "billing-service requires --check or --serve")
		os.Exit(2)
	}
	if err := serveBilling(*addr, *configPath); err != nil {
		fmt.Fprintf(os.Stderr, "billing-service failed: %v\n", err)
		os.Exit(1)
	}
}

func checkRuntime() error {
	server := grpc.NewServer()
	defer server.Stop()
	runtime, err := billingruntime.New(noopBillingServer{})
	if err != nil {
		return err
	}
	return runtime.RegisterGRPC(server)
}

func serveBilling(addr, configPath string) error {
	if err := platformserver.ValidateInternalToken(); err != nil {
		return fmt.Errorf("gRPC internal token validation: %w", err)
	}
	dsn := strings.TrimSpace(os.Getenv("MYSQL_DSN"))
	if dsn == "" {
		return errors.New("MYSQL_DSN is required")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return fmt.Errorf("connect mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	metricsServer, err := platformserver.StartMetricsServer(envOrDefault("METRICS_ADDR", ""))
	if err != nil {
		return err
	}
	defer platformserver.ShutdownMetricsServer(context.Background(), metricsServer)
	repo, err := persistence.NewGormRepository(db)
	if err != nil {
		return err
	}
	policy, err := persistence.NewEntitlementPolicy(db)
	if err != nil {
		return err
	}
	mode, alipayConfig, alipayRequired, err := loadBillingConfiguration(configPath)
	if err != nil {
		return err
	}
	billing, err := service.NewBilling(repo, policy, mode)
	if err != nil {
		return err
	}
	var alipay service.AlipayGateway
	configured := strings.TrimSpace(alipayConfig.AppID) != "" || strings.TrimSpace(alipayConfig.PrivateKey) != "" || strings.TrimSpace(alipayConfig.VerifyPublicKey) != "" || strings.TrimSpace(alipayConfig.SellerID) != ""
	if configured || alipayRequired {
		alipay, err = payment.NewAlipay(alipayConfig)
		if err != nil {
			return fmt.Errorf("initialize Alipay sandbox: %w", err)
		}
	} else {
		alipay = payment.NewUnavailableAlipay(errors.New("Alipay sandbox credentials are not configured"))
	}
	commerce, err := service.NewCommerce(db, alipay)
	if err != nil {
		return err
	}
	api, err := billinggrpc.NewServer(billing, commerce)
	if err != nil {
		return err
	}
	runtime, err := billingruntime.New(api)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	options := []grpc.ServerOption{grpc.ChainUnaryInterceptor(platformserver.UnaryAuthInterceptor())}
	if tlsOption, enabled, tlsErr := platformserver.TransportSecurityOption(os.Getenv("GRPC_TLS_CERT_FILE"), os.Getenv("GRPC_TLS_KEY_FILE")); tlsErr != nil {
		return tlsErr
	} else if enabled {
		options = append(options, tlsOption)
	}
	server := grpc.NewServer(options...)
	if err := runtime.RegisterGRPC(server); err != nil {
		return err
	}
	healthpb.RegisterHealthServer(server, &billingHealthServer{base: platformserver.NewHealthServer(sqlDB, nil, nil), paymentReady: configured})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	go maintainBilling(ctx, repo, commerce)
	go func() { <-ctx.Done(); server.GracefulStop() }()
	fmt.Fprintf(os.Stdout, "billing grpc server listening on %s in %s mode\n", listener.Addr(), mode)
	if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}
	return nil
}

func maintainBilling(ctx context.Context, repo *persistence.GormRepository, commerce *service.Commerce) {
	// Recover payments that became due while the service was stopped before
	// entering the periodic cadence. This keeps restart recovery bounded by the
	// Alipay query latency instead of adding another five-minute delay.
	if err := commerce.ReconcilePendingPayments(ctx, 100); err != nil {
		fmt.Fprintf(os.Stderr, "reconcile pending Alipay payments on startup: %v\n", err)
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	cycles := 0
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if err := repo.RunMaintenance(ctx, now.UTC()); err != nil {
				fmt.Fprintf(os.Stderr, "billing maintenance: %v\n", err)
			}
			if err := commerce.UpdateOperationalMetrics(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "update billing operational metrics: %v\n", err)
			}
			cycles++
			if cycles%5 == 0 {
				if err := commerce.ReconcilePendingPayments(ctx, 100); err != nil {
					fmt.Fprintf(os.Stderr, "reconcile pending Alipay payments: %v\n", err)
				}
			}
		}
	}
}

type noopBillingServer struct {
	pb.UnimplementedBillingServiceServer
}

type billingHealthServer struct {
	healthpb.UnimplementedHealthServer
	base         *platformserver.HealthServer
	paymentReady bool
}

func (s *billingHealthServer) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	response, err := s.base.Check(ctx, req)
	if err != nil {
		return response, err
	}
	if !s.paymentReady {
		response.Status = healthpb.HealthCheckResponse_NOT_SERVING
	}
	return response, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func loadBillingConfiguration(path string) (model.EnforcementMode, payment.AlipayConfig, bool, error) {
	if strings.TrimSpace(path) != "" {
		cfg, err := billingconfig.Load(path)
		if err != nil {
			return "", payment.AlipayConfig{}, false, err
		}
		return model.EnforcementMode(cfg.Billing.Mode), cfg.PaymentConfig(), cfg.Alipay.Required, nil
	}
	return model.EnforcementMode(envOrDefault("AI_BILLING_MODE", string(model.ModeShadow))), payment.AlipayConfig{
		Environment: envOrDefault("ALIPAY_ENV", "sandbox"), GatewayURL: envOrDefault("ALIPAY_GATEWAY_URL", payment.DefaultSandboxGateway),
		AppID: os.Getenv("ALIPAY_APP_ID"), PrivateKey: os.Getenv("ALIPAY_PRIVATE_KEY"), VerifyPublicKey: os.Getenv("ALIPAY_VERIFY_PUBLIC_KEY"),
		SellerID: os.Getenv("ALIPAY_SELLER_ID"), NotifyURL: os.Getenv("ALIPAY_NOTIFY_URL"), ReturnURL: os.Getenv("ALIPAY_RETURN_URL"),
		DesktopEnabled: envBool("ALIPAY_DESKTOP_ENABLED", true), WAPEnabled: envBool("ALIPAY_WAP_ENABLED", true),
	}, strings.EqualFold(strings.TrimSpace(os.Getenv("ALIPAY_REQUIRED")), "true"), nil
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
