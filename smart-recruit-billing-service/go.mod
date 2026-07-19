module smart-recruit-billing-service

go 1.25.5

require (
	github.com/google/uuid v1.6.0
	google.golang.org/grpc v1.73.0
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.30.0
	smart-recruit-proto v0.0.0
	smart-recruit-platform-go v0.0.0
)

require (
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

replace smart-recruit-proto => ../smart-recruit-proto

replace smart-recruit-platform-go => ../smart-recruit-platform-go
