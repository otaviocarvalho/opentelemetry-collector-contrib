package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/lokiexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/otel/metric/nonrecording"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func main() {
	logs := plog.NewLogs()
	rs := logs.ResourceLogs().AppendEmpty()
	rs.Resource().Attributes().InsertString("instance", "my-service")

	sl := rs.ScopeLogs().AppendEmpty()

	logRecord := sl.LogRecords().AppendEmpty()
	logRecord.Body().SetStringVal("Adding a test log entry from a local binary: " + uuid.New().String())
	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	f := lokiexporter.NewFactory()
	c := f.CreateDefaultConfig().(*lokiexporter.Config)
	c.Labels.ResourceAttributes = map[string]string{
		"instance": "",
	}
	c.Endpoint = "https://logs-prod3.grafana.net/loki/api/v1/push"
	c.Headers = map[string]string{
		// Create token: `echo -n "user:password" | base64`
		"Authorization": "Basic <insert-token-here>",
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Printf("failed to create a logger: %v\n", err)
		return
	}

	exp, err := f.CreateLogsExporter(context.Background(), component.ExporterCreateSettings{
		TelemetrySettings: component.TelemetrySettings{
			Logger:         logger,
			TracerProvider: trace.NewNoopTracerProvider(),
			MeterProvider:  nonrecording.NewNoopMeterProvider(),
			MetricsLevel:   configtelemetry.LevelNone,
		},
	}, c)
	if err != nil {
		fmt.Printf("failed to create the Loki exporter: %v\n", err)
		return
	}

	err = exp.Start(context.Background(), componenttest.NewNopHost())
	if err != nil {
		fmt.Printf("failed to start the Loki exporter: %v\n", err)
		return
	}

	if err := exp.ConsumeLogs(context.Background(), logs); err != nil {
		fmt.Printf("failed to send data to Loki: %v\n", err)
		return
	}

	err = exp.Shutdown(context.Background())
	if err != nil {
		fmt.Printf("failed to shutdown the Loki exporter: %v\n", err)
		return
	}

	fmt.Println("everything seems to have worked just fine")

}
