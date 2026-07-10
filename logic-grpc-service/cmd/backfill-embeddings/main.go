package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"logic-grpc-service/pkg/crypto"
	"logic-grpc-service/repository"
	"logic-grpc-service/service"
)

func main() {
	dsn := flag.String("dsn", "root:Aa123456@tcp(127.0.0.1:3306)/recruitment?charset=utf8mb4&parseTime=True&loc=Local", "MySQL DSN")
	objectType := flag.String("object-type", "", "object type to backfill (agent_skill, ai_memory, empty=all)")
	limit := flag.Int("limit", 0, "max number of objects to process (0=unlimited)")
	batchSize := flag.Int("batch-size", 20, "batch size for progress reporting")
	force := flag.Bool("force", false, "re-process even if already ready")
	dryRun := flag.Bool("dry-run", false, "dry run (no writes)")
	modelID := flag.Int64("model-id", 0, "specific embedding model ID (0=default)")
	flag.Parse()

	db, err := gorm.Open(mysql.Open(*dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(5)
	defer sqlDB.Close()

	embeddingRepo := repository.NewAIEmbeddingRepo(db)
	embeddingSvc := service.NewEmbeddingService(embeddingRepo, nil, crypto.EncryptionKey{})

	skillRepo := repository.NewAgentSkillRepo(db)
	memoryRepo := repository.NewMemoryRepo(db)

	backfillSvc := service.NewEmbeddingBackfillService(db, embeddingSvc, skillRepo, memoryRepo)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fmt.Println("=== Embedding Backfill ===")
	fmt.Printf("  object_type: %s\n", *objectType)
	fmt.Printf("  limit: %d\n", *limit)
	fmt.Printf("  batch_size: %d\n", *batchSize)
	fmt.Printf("  force: %v\n", *force)
	fmt.Printf("  dry_run: %v\n", *dryRun)
	fmt.Println()

	result, err := backfillSvc.Run(ctx, service.BackfillInput{
		ObjectType: *objectType,
		Limit:      *limit,
		BatchSize:  *batchSize,
		Force:      *force,
		DryRun:     *dryRun,
		ModelID:    *modelID,
	})
	if err != nil {
		log.Fatalf("backfill failed: %v", err)
	}

	fmt.Println()
	fmt.Println("=== Backfill Complete ===")
	fmt.Printf("  success: %d\n", result.SuccessCount)
	fmt.Printf("  failed:  %d\n", result.FailedCount)
	fmt.Printf("  skipped: %d\n", result.SkippedCount)
	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println("Errors:")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}
}
