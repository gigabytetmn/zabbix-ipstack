/*
** Copyright (C) 2001-2025 Zabbix SIA
**
** Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
** documentation files (the "Software"), to deal in the Software without restriction, including without limitation the
** rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
** permit persons to whom the Software is furnished to do so, subject to the following conditions:
**
** The above copyright notice and this permission notice shall be included in all copies or substantial portions
** of the Software.
**
** THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
** WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
** COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
** TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
** SOFTWARE.
**/

package plugin

import (
	"context"
	"time"

	"golang.zabbix.com/plugin/example/plugin/handlers"
	"golang.zabbix.com/plugin/example/plugin/params"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/metric"
	"golang.zabbix.com/sdk/plugin"
	"golang.zabbix.com/sdk/plugin/container"
	"golang.zabbix.com/sdk/zbxerr"
)

const (
	// Name of the plugin.
	Name = "Example"

	myIPMetric  = exampleMetricKey("example.myip")
	goEnvMetric = exampleMetricKey("example.go.env")
)

var (
	_ plugin.Configurator = (*examplePlugin)(nil)
	_ plugin.Exporter     = (*examplePlugin)(nil)
	_ plugin.Runner       = (*examplePlugin)(nil)
)

type exampleMetricKey string

type exampleMetric struct {
	metric  *metric.Metric
	handler handlers.HandlerFunc
}

type examplePlugin struct {
	plugin.Base
	config  *pluginConfig
	metrics map[exampleMetricKey]*exampleMetric
}

// Launch launches the Example plugin. Blocks until plugin execution has
// finished.
func Launch() error {
	p := &examplePlugin{}

	err := p.registerMetrics()
	if err != nil {
		return err
	}

	h, err := container.NewHandler(Name)
	if err != nil {
		return errs.Wrap(err, "failed to create new handler")
	}

	p.Logger = h

	err = h.Execute()
	if err != nil {
		return errs.Wrap(err, "failed to execute plugin handler")
	}

	return nil
}

// Start starts the example plugin. Is required for plugin to match runner interface.
func (p *examplePlugin) Start() {
	p.Logger.Infof("Start called")
}

// Stop stops the example plugin. Is required for plugin to match runner interface.
func (p *examplePlugin) Stop() {
	p.Logger.Infof("Stop called")
}

// Export collects all the metrics.
func (p *examplePlugin) Export(key string, rawParams []string, _ plugin.ContextProvider) (any, error) {
	m, ok := p.metrics[exampleMetricKey(key)]
	if !ok {
		return nil, errs.Wrapf(zbxerr.ErrorUnsupportedMetric, "unknown metric %q", key)
	}

	metricParams, extraParams, hardcodedParams, err := m.metric.EvalParams(rawParams, p.config.Sessions)
	if err != nil {
		return nil, errs.Wrap(err, "failed to evaluate metric parameters")
	}

	err = metric.SetDefaults(metricParams, hardcodedParams, p.config.Default)
	if err != nil {
		return nil, errs.Wrap(err, "failed to set default params")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(p.config.Timeout)*time.Second,
	)
	defer cancel()

	res, err := m.handler(ctx, metricParams, extraParams...)
	if err != nil {
		return nil, errs.Wrap(err, "failed to execute handler")
	}

	return res, nil
}

func (p *examplePlugin) registerMetrics() error {
	handler := handlers.New()

	p.metrics = map[exampleMetricKey]*exampleMetric{
		myIPMetric: {
			metric: metric.New(
				"Returns the availability groups.",
				params.Params,
				false,
			),
			handler: handlers.WithJSONResponse(
				handlers.WithCredentialValidation(
					handler.MyIP,
				),
			),
		},
		goEnvMetric: {
			metric: metric.New(
				"Returns the result rows of a custom query.",
				params.Params,
				true,
			),
			handler: handlers.WithJSONResponse(
				handlers.WithCredentialValidation(handler.GoEnvironment),
			),
		},
	}

	metricSet := metric.MetricSet{}

	for k, m := range p.metrics {
		metricSet[string(k)] = m.metric
	}

	err := plugin.RegisterMetrics(p, Name, metricSet.List()...)
	if err != nil {
		return errs.Wrap(err, "failed to register metrics")
	}

	return nil
}
