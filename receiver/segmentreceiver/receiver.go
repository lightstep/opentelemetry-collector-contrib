package segmentreceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.uber.org/zap"
	segmentAnalytics "gopkg.in/segmentio/analytics-go.v3"
)

type segmentReceiver struct {
	mu     sync.Mutex
	logger *zap.Logger
	config *Config

	host         component.Host
	nextConsumer consumer.Metrics
	instanceName string
	server       *http.Server

	startOnce  sync.Once
	stopOnce   sync.Once
	shutdownWG sync.WaitGroup
}

var errNextConsumerRespBody = []byte(`"Internal Server Error"`)

var _ http.Handler = (*segmentReceiver)(nil)

func New(
	logger *zap.Logger,
	config Config,
	nextConsumer consumer.Metrics,
) (*segmentReceiver, error) {
	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}

	if config.HTTPServerSettings.Endpoint == "" {
		config.HTTPServerSettings.Endpoint = defaultBindEndpoint // TODO: check this
	}

	r := &segmentReceiver{
		logger:       logger,
		config:       &config,
		nextConsumer: nextConsumer,
		instanceName: config.Name(),
	}
	return r, nil
}

func (r *segmentReceiver) Start(_ context.Context, host component.Host) error {
	if host == nil {
		return errors.New("nil host")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	err := componenterror.ErrAlreadyStarted
	r.startOnce.Do(func() {
		err = nil
		r.host = host
		r.server = r.config.HTTPServerSettings.ToServer(r)
		listener, err := r.config.HTTPServerSettings.ToListener()
		if err != nil {
			return
		}
		r.shutdownWG.Add(1)
		go func() {
			defer r.shutdownWG.Done()
			if errHTTP := r.server.Serve(listener); errHTTP != http.ErrServerClosed {
				host.ReportFatalError(errHTTP)
			}
		}()
	})

	return err
}

func (r *segmentReceiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if c, ok := client.FromHTTP(req); ok {
		ctx = client.NewContext(ctx, c)
	}

	transportType := req.Header.Get("Content-Type")
	ctx = obsreport.ReceiverContext(ctx, r.instanceName, transportType)
	ctx = obsreport.StartMetricsReceiveOp(ctx, r.instanceName, transportType)

	slurp, _ := ioutil.ReadAll(req.Body)
	if c, ok := req.Body.(io.Closer); ok {
		_ = c.Close()
	}
	_ = req.Body.Close()

	var md pdata.Metrics
	var err error
	md, err = r.segmentEventToMetric(slurp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	consumerErr := r.nextConsumer.ConsumeMetrics(ctx, md)
	obsreport.EndMetricsReceiveOp(ctx, "segment", 1, consumerErr)

	if consumerErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errNextConsumerRespBody)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (r *segmentReceiver) segmentEventToMetric(blob []byte) (metrics pdata.Metrics, err error) {
	var track segmentAnalytics.Track
	track, err = r.deserializeFromJSON(blob)
	if track.Event == "" {
		return pdata.NewMetrics(), err
	}

	// Map to the metric
	dp := pdata.NewIntDataPoint()
	dp.SetValue(1)
	dp.SetTimestamp(pdata.TimestampFromTime(track.Timestamp))
	dp.LabelsMap().Insert("user_id", track.UserId)
	for k, v := range track.Properties {
		dp.LabelsMap().Insert(k, fmt.Sprintf("%v", v))
	}
	nm := pdata.NewMetric()
	nm.SetName(track.Event)
	nm.SetUnit("count")
	nm.SetDataType(pdata.MetricDataTypeIntGauge)
	nm.IntGauge().DataPoints().Append(dp)

	ilm := pdata.NewInstrumentationLibraryMetrics()
	ilm.Metrics().Append(nm)

	mts := pdata.NewMetrics()
	mts.ResourceMetrics().Resize(1)
	mts.ResourceMetrics().At(0).InstrumentationLibraryMetrics().Resize(0)
	mts.ResourceMetrics().At(0).InstrumentationLibraryMetrics().Append(ilm)

	return mts, nil
}

func (r *segmentReceiver) deserializeFromJSON(jsonBlob []byte) (t segmentAnalytics.Track, err error) {
	if err = json.Unmarshal(jsonBlob, &t); err != nil {
		return segmentAnalytics.Track{}, err
	}
	return t, nil
}

func (r *segmentReceiver) Shutdown(context.Context) error {
	err := componenterror.ErrAlreadyStopped
	r.stopOnce.Do(func() {
		err = r.server.Close()
		r.shutdownWG.Wait()
	})
	return err
}
