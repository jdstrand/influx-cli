package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/influxdata/influx-cli/v2/pkg/cli/middleware"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func TestApp_HostSpecificErrors(t *testing.T) {
	tests := []struct {
		name          string
		commandMw     cli.BeforeFunc
		svrBuild      string
		svrResCode    int
		wantErrString string
	}{
		{
			name:          "OSS command on Cloud host - with error",
			commandMw:     middleware.OSSOnly,
			svrBuild:      middleware.CloudBuildHeader,
			svrResCode:    http.StatusServiceUnavailable,
			wantErrString: fmt.Sprintf("%s\n", middleware.WrongHostErrString(middleware.OSSBuildHeader, middleware.CloudBuildHeader)),
		},
		{
			name:          "Cloud command on OSS host - with error",
			commandMw:     middleware.CloudOnly,
			svrBuild:      middleware.OSSBuildHeader,
			svrResCode:    http.StatusServiceUnavailable,
			wantErrString: fmt.Sprintf("%s\n", middleware.WrongHostErrString(middleware.CloudBuildHeader, middleware.OSSBuildHeader)),
		},
		{
			name:          "OSS command on OSS host - with error",
			commandMw:     middleware.OSSOnly,
			svrBuild:      middleware.OSSBuildHeader,
			svrResCode:    http.StatusServiceUnavailable,
			wantErrString: fmt.Sprintf("%s\n", "Error: health check failed: 503 Service Unavailable: unavailable"),
		},
		{
			name:          "Cloud command on Cloud host - with error",
			commandMw:     middleware.CloudOnly,
			svrBuild:      middleware.CloudBuildHeader,
			svrResCode:    http.StatusServiceUnavailable,
			wantErrString: fmt.Sprintf("%s\n", "Error: health check failed: 503 Service Unavailable: unavailable"),
		},
		{
			name:          "OSS command on Cloud host - no error",
			commandMw:     middleware.OSSOnly,
			svrBuild:      middleware.CloudBuildHeader,
			svrResCode:    http.StatusOK,
			wantErrString: "",
		},
		{
			name:          "Cloud command on OSS host - no error",
			commandMw:     middleware.CloudOnly,
			svrBuild:      middleware.OSSBuildHeader,
			svrResCode:    http.StatusOK,
			wantErrString: "",
		},
		{
			name:          "OSS command on OSS host - no error",
			commandMw:     middleware.OSSOnly,
			svrBuild:      middleware.OSSBuildHeader,
			svrResCode:    http.StatusOK,
			wantErrString: "",
		},
		{
			name:          "Cloud command on Cloud host - no error",
			commandMw:     middleware.CloudOnly,
			svrBuild:      middleware.CloudBuildHeader,
			svrResCode:    http.StatusOK,
			wantErrString: "",
		},
		{
			name:          "Non-specific command on OSS host - with error",
			commandMw:     nil,
			svrBuild:      middleware.OSSBuildHeader,
			svrResCode:    http.StatusServiceUnavailable,
			wantErrString: fmt.Sprintf("%s\n", "Error: health check failed: 503 Service Unavailable: unavailable"),
		},
		{
			name:          "Non-specific command on Cloud host - with error",
			commandMw:     nil,
			svrBuild:      middleware.CloudBuildHeader,
			svrResCode:    http.StatusServiceUnavailable,
			wantErrString: fmt.Sprintf("%s\n", "Error: health check failed: 503 Service Unavailable: unavailable"),
		},
		{
			name:          "Non-specific command on OSS host - no error",
			commandMw:     nil,
			svrBuild:      middleware.OSSBuildHeader,
			svrResCode:    http.StatusOK,
			wantErrString: "",
		},
		{
			name:          "Non-specific command on Cloud host - no error",
			commandMw:     nil,
			svrBuild:      middleware.CloudBuildHeader,
			svrResCode:    http.StatusOK,
			wantErrString: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli.OsExiter = func(code int) {
				require.Equal(t, 1, code)
			}
			var b bytes.Buffer
			errWriter := &testWriter{
				b: &b,
			}
			cli.ErrWriter = errWriter

			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("X-Influxdb-Build", tt.svrBuild)
				w.WriteHeader(tt.svrResCode)
				_, err := fmt.Fprintf(w, `{"message":%q}`, "unavailable")
				require.NoError(t, err)
			}))
			defer svr.Close()
			cmd := newPingCmd()
			if tt.commandMw != nil {
				cmd = middleware.AddMWToCmds([]cli.Command{cmd}, tt.commandMw)[0]
			}
			app := newApp()
			app.Commands = []cli.Command{cmd}

			args := []string{
				"influx",
				"ping",
				"--host",
				svr.URL,
			}

			_ = app.Run(args)
			require.Equal(t, tt.wantErrString, errWriter.b.String())
			if tt.wantErrString == "" {
				require.False(t, errWriter.written)
			}
		})
	}
}

type testWriter struct {
	b       *bytes.Buffer
	written bool
}

func (w *testWriter) Write(p []byte) (int, error) {
	w.written = true
	return w.b.Write(p)
}
