package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"logic-grpc-service/ai"
	"logic-grpc-service/repository"
)

func main() {
	os.Setenv("GORM_LOGGER_LEVEL", "warn")

	db, err := gorm.Open(mysql.Open("root:Aa123456@tcp(127.0.0.1:3306)/recruitment?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
	if err != nil {
		log.Fatal("connect db:", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(5)
	defer sqlDB.Close()

	appRepo := repository.NewApplicationRepo(db)
	jobRepo := repository.NewJobRepo(db)
	resumeRepo := repository.NewResumeRepo(db)
	executor := ai.NewToolExecutor(appRepo, jobRepo, resumeRepo, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test 1: HR=2, list all applications
	fmt.Println("===== list_all_applications (HR=2, page=1, page_size=10) =====")
	result, err := executor.Execute(ctx, 2, "list_all_applications", map[string]any{"page": float64(1), "page_size": float64(10)})
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("result_bytes: %d\n", len(result.Content))
		var pretty map[string]any
		if err := json.Unmarshal([]byte(result.Content), &pretty); err == nil {
			b, _ := json.MarshalIndent(pretty, "", "  ")
			fmt.Println(string(b))
		} else {
			fmt.Println(result.Content)
		}
	}

	// Test 2: HR=2, total
	fmt.Println("\n===== query_total_applications (HR=2) =====")
	result, err = executor.Execute(ctx, 2, "query_total_applications", nil)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Println(result.Content)
	}

	// Test 3: HR=2, search candidates
	fmt.Println("\n===== search_candidates (HR=2, keyword=张三) =====")
	result, err = executor.Execute(ctx, 2, "search_candidates", map[string]any{"keyword": "张三"})
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("result_bytes: %d\n", len(result.Content))
		fmt.Println(result.Content)
	}

	// Test 4: Check how many applications exist
	fmt.Println("\n===== raw count =====")
	var count int64
	db.Table("applications").Joins("JOIN jobs ON jobs.id = applications.job_id").Where("jobs.hr_id = ?", 2).Count(&count)
	fmt.Printf("total applications for HR=2: %d\n", count)

	fmt.Println("\n===== raw data (first 5 applications for HR=2) =====")
	var rows []map[string]any
	db.Raw(`SELECT applications.id, applications.user_id, candidate_profiles.real_name,
		candidate_profiles.phone, jobs.title, applications.status
		FROM applications
		JOIN jobs ON jobs.id = applications.job_id
		LEFT JOIN candidate_profiles ON candidate_profiles.user_id = applications.user_id
		WHERE jobs.hr_id = ? LIMIT 5`, 2).Scan(&rows)
	for _, row := range rows {
		bs, _ := json.Marshal(row)
		fmt.Println(string(bs))
	}
}
