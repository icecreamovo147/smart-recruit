package oss

import (
	"errors"
	"net/http"
	"testing"

	aliyunoss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func TestIsNotFound_Nil(t *testing.T) {
	if IsNotFound(nil) {
		t.Error("nil error should not be considered not-found")
	}
}

func TestIsNotFound_AliyunNoSuchKey(t *testing.T) {
	err := aliyunoss.ServiceError{Code: "NoSuchKey", StatusCode: http.StatusNotFound}
	if !IsNotFound(err) {
		t.Error("aliyun NoSuchKey error should be detected as not-found")
	}
}

func TestIsNotFound_GenericError(t *testing.T) {
	err := errors.New("some unrelated error")
	if IsNotFound(err) {
		t.Error("generic error should not be considered not-found")
	}
}
