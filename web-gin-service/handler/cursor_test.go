package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type publicCursorJobClient struct {
	got *pb.ListPublicJobsRequest
}

func (f *publicCursorJobClient) CreateJob(context.Context, *pb.CreateJobRequest, ...grpc.CallOption) (*pb.CreateJobResponse, error) {
	return nil, nil
}

func (f *publicCursorJobClient) UpdateJob(context.Context, *pb.UpdateJobRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return nil, nil
}

func (f *publicCursorJobClient) OfflineJob(context.Context, *pb.OfflineJobRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return nil, nil
}

func (f *publicCursorJobClient) OnlineJob(context.Context, *pb.OfflineJobRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return nil, nil
}

func (f *publicCursorJobClient) ListHRJobs(context.Context, *pb.ListHRJobsRequest, ...grpc.CallOption) (*pb.ListJobsResponse, error) {
	return nil, nil
}

func (f *publicCursorJobClient) ListPublicJobs(_ context.Context, req *pb.ListPublicJobsRequest, _ ...grpc.CallOption) (*pb.ListJobsResponse, error) {
	copied := *req
	f.got = &copied
	return &pb.ListJobsResponse{Code: 0, Msg: "success"}, nil
}

func (f *publicCursorJobClient) GetJobDetail(context.Context, *pb.GetJobDetailRequest, ...grpc.CallOption) (*pb.GetJobDetailResponse, error) {
	return nil, nil
}

func (f *publicCursorJobClient) ListJobOptions(context.Context, *pb.ListJobOptionsRequest, ...grpc.CallOption) (*pb.ListJobOptionsResponse, error) {
	return nil, nil
}

func (f *publicCursorJobClient) ListDepartmentLocations(context.Context, *pb.ListDepartmentLocationsRequest, ...grpc.CallOption) (*pb.ListDepartmentLocationsResponse, error) {
	return nil, nil
}

type cursorNotificationClient struct {
	got *pb.ListNotificationsRequest
}

func (f *cursorNotificationClient) ListNotifications(_ context.Context, req *pb.ListNotificationsRequest, _ ...grpc.CallOption) (*pb.ListNotificationsResponse, error) {
	copied := *req
	f.got = &copied
	return &pb.ListNotificationsResponse{Code: 0, Msg: "success"}, nil
}

func (f *cursorNotificationClient) UnreadNotificationCount(context.Context, *pb.UnreadNotificationCountRequest, ...grpc.CallOption) (*pb.UnreadNotificationCountResponse, error) {
	return nil, nil
}

func (f *cursorNotificationClient) NotificationSummary(context.Context, *pb.NotificationSummaryRequest, ...grpc.CallOption) (*pb.NotificationSummaryResponse, error) {
	return nil, nil
}

func (f *cursorNotificationClient) MarkNotificationRead(context.Context, *pb.MarkNotificationReadRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return nil, nil
}

func (f *cursorNotificationClient) MarkAllNotificationsRead(context.Context, *pb.MarkAllNotificationsReadRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return nil, nil
}

func TestPublicListJobsPassesCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &publicCursorJobClient{}
	handler := NewPublicHandler(&rpc.Clients{Job: fake})
	router := gin.New()
	router.GET("/jobs", handler.ListJobs)

	req := httptest.NewRequest(http.MethodGet, "/jobs?page=3&page_size=25&keyword=go&cursor=abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if fake.got == nil {
		t.Fatalf("expected ListPublicJobs to be called")
	}
	if fake.got.Cursor != "abc" || fake.got.Page != 0 || fake.got.PageSize != 25 || fake.got.Keyword != "go" {
		t.Fatalf("unexpected request: %+v", fake.got)
	}
}

func TestNotificationListPassesCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &cursorNotificationClient{}
	handler := NewNotificationHandler(&rpc.Clients{Notification: fake}, nil)
	router := gin.New()
	router.GET("/notifications", func(c *gin.Context) {
		c.Set("user_id", int64(7))
		c.Set("account_type", "candidate")
		handler.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/notifications?page=2&page_size=30&cursor=next", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if fake.got == nil {
		t.Fatalf("expected ListNotifications to be called")
	}
	if fake.got.UserId != 7 || fake.got.AccountType != "candidate" || fake.got.Cursor != "next" || fake.got.Page != 0 || fake.got.PageSize != 30 {
		t.Fatalf("unexpected request: %+v", fake.got)
	}
}

func TestNotificationListPassesAccountTypeForStaff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &cursorNotificationClient{}
	handler := NewNotificationHandler(&rpc.Clients{Notification: fake}, nil)
	router := gin.New()
	router.GET("/notifications", func(c *gin.Context) {
		c.Set("user_id", int64(7))
		c.Set("account_type", "staff")
		handler.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/notifications?page=1&page_size=20", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if fake.got == nil {
		t.Fatalf("expected ListNotifications to be called")
	}
	if fake.got.UserId != 7 || fake.got.AccountType != "staff" {
		t.Fatalf("unexpected notification receiver: user_id=%d account_type=%s", fake.got.UserId, fake.got.AccountType)
	}
}
