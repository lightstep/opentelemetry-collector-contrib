// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lightstepexporter

import (
	"context"
	"net/url"
	"strconv"

	"github.com/lightstep/lightstep-tracer-go/lightstepoc"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exporterhelper"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	spandatatranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace/spandata"
)

type lightStepExporter struct {
	exporter *lightstepoc.Exporter
}

func (e *lightStepExporter) exportTrace(ctx context.Context, td consumerdata.TraceData) (int, error) {
	var errs []error
	for _, span := range td.Spans {
		sd, err := spandatatranslator.ProtoSpanToOCSpanData(span)
		if err == nil {
			e.exporter.ExportSpan(sd)
		} else {
			errs = append(errs, err)
		}
	}
	return len(errs), oterr.CombineErrors(errs)
}

func (e *lightStepExporter) shutdown() error {
	e.exporter.Close(context.Background())
	return nil
}

func newTraceExporter(cfg *Config) (exporter.TraceExporter, error) {
	u, err := url.Parse(cfg.SatelliteURL)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return nil, err
	}

	insecure := false
	if u.Scheme == "http" {
		insecure = true
	}

	exporterOptions := []lightstepoc.Option{
		lightstepoc.WithAccessToken(cfg.AccessToken),
		lightstepoc.WithSatelliteHost(u.Host),
		lightstepoc.WithSatellitePort(port),
		lightstepoc.WithInsecure(insecure),
	}

	exporter, err := lightstepoc.NewExporter(exporterOptions...)
	if err != nil {
		return nil, err
	}
	lsExporter := lightStepExporter{
		exporter: exporter,
	}
	return exporterhelper.NewTraceExporter(
		cfg,
		lsExporter.exportTrace,
		exporterhelper.WithTracing(true),
		exporterhelper.WithMetrics(true),
		exporterhelper.WithShutdown(lsExporter.shutdown),
	)
}
