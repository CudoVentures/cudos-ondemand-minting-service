package email

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go"

	"github.com/stretchr/testify/require"
)

func TestSendgridEmailService(t *testing.T) {
	defer srv.Close()
	log.Logger = zerolog.New(logsOutput)

	for _, testCase := range []struct {
		name         string
		service      *sendgridEmailService
		expectedLogs string
	}{
		{
			name:         "send email successfully",
			service:      newEmailService("", 0),
			expectedLogs: `{"level":"info","message":"sending email succeeded"}`,
		},
		{
			name:         "send email with invalid config",
			service:      &sendgridEmailService{},
			expectedLogs: `{"level":"warn","message":"sending email failed: invalid config"}`,
		},
		{
			name:         "send email before interval",
			service:      newEmailService("", 1*time.Minute),
			expectedLogs: `{"level":"warn","message":"sending email failed: waiting for the next email send interval"}`,
		},
		{
			name:         "send email with invalid request method",
			service:      newEmailService("?", 0),
			expectedLogs: `{"level":"warn","error":"sending email failed: net/http: invalid method \"?\""}`,
		},
		{
			name:         "send email with invalid request params",
			service:      newEmailService("invalid", 0),
			expectedLogs: `{"level":"warn","error":"sending email failed: invalid"}`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			defer logsOutput.Reset()

			testCase.service.SendEmail(str)

			actualLogs := strings.ReplaceAll(logsOutput.String(), "\n", "")
			require.Equal(t, testCase.expectedLogs, actualLogs)

			if strings.Contains(testCase.expectedLogs, `"level":"info"`) {
				require.True(t, testCase.service.nextEmailSendTime.After(time.Now()))
			} else {
				require.False(t, testCase.service.nextEmailSendTime.After(time.Now().Add(1*time.Minute)))
			}
		})
	}

	t.Run("new service", func(t *testing.T) {
		defer logsOutput.Reset()

		expectedService := &sendgridEmailService{
			client:            sendgrid.NewSendClient(str),
			sender:            str,
			recipient:         str,
			emailSendInterval: time.Minute * 30,
		}
		service := NewSendgridEmailService(config.Config{
			EmailFrom:         str,
			ServiceEmail:      str,
			SendgridApiKey:    str,
			EmailSendInterval: time.Minute * 30,
		})
		require.Empty(t, logsOutput.String())
		require.Equal(t, time.Now().Year(), service.nextEmailSendTime.Year())
		service.nextEmailSendTime = time.Time{}
		require.Equal(t, expectedService, service)
	})

	t.Run("new service with invalid config", func(t *testing.T) {
		defer logsOutput.Reset()

		service := NewSendgridEmailService(config.Config{})
		expectedLogs := "{\"level\":\"warn\",\"message\":\"creating email service failed: invalid config\"}"
		actualLogs := strings.ReplaceAll(logsOutput.String(), "\n", "")
		require.Equal(t, expectedLogs, actualLogs)
		require.Empty(t, service)
	})
}

func newEmailService(method string, sendEmailInterval time.Duration) *sendgridEmailService {
	return &sendgridEmailService{
		client: &sendgrid.Client{Request: rest.Request{
			BaseURL: srv.URL,
			Method:  rest.Method(method),
		}},
		nextEmailSendTime: time.Now().Add(sendEmailInterval),
		emailSendInterval: 30 * time.Minute,
	}
}

var srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.Method == "invalid" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid"))
	} else {
		w.WriteHeader(http.StatusOK)
	}
}))

var logsOutput = &bytes.Buffer{}

const str = "test"
