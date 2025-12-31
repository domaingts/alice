package log_test

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/xtls/xray-core/app/log"
	"github.com/xtls/xray-core/common"
	clog "github.com/xtls/xray-core/common/log"
	"github.com/xtls/xray-core/testing/mocks"
)

func TestCustomLogHandler(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	var loggedValue []string

	mockHandler := mocks.NewLogHandler(mockCtl)
	mockHandler.EXPECT().Handle(gomock.Any()).AnyTimes().DoAndReturn(func(msg clog.Message) {
		loggedValue = append(loggedValue, msg.String())
	})

	log.RegisterHandlerCreator(log.LogType_Console, func(lt log.LogType, options log.HandlerCreatorOptions) (clog.Handler, error) {
		return mockHandler, nil
	})

	logger, err := log.New(context.Background(), &log.Config{
		ErrorLogLevel: clog.Severity_Debug,
		ErrorLogType:  log.LogType_Console,
		AccessLogType: log.LogType_None,
	})
	common.Must(err)

	common.Must(logger.Start())

	clog.Record(&clog.GeneralMessage{
		Severity: clog.Severity_Debug,
		Content:  "test",
	})

	if len(loggedValue) < 2 {
		t.Fatal("expected 2 log messages, but actually ", loggedValue)
	}

	if loggedValue[1] != "[Debug] test" {
		t.Fatal("expected '[Debug] test', but actually ", loggedValue[1])
	}

	common.Must(logger.Close())
}

func TestParse(t *testing.T) {
	ip := "111:111:123::"
	dot1 := strings.IndexByte(ip, ':')
	if dot1 < 0 {
		panic("no dot")
	}
	t.Log(dot1, ip[:dot1])
	dot2 := strings.IndexByte(ip[dot1+1:], ':')
	if dot2 < 0 {
		panic("not dot2")
	}
	t.Log(ip[:dot1+dot2+1]+".*.*")
}
