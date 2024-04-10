// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sumconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/sumconnector"

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil"
)

var noAttributes = [16]byte{}

func newCounter[K any](metricDefs map[string]metricDef[K]) *counter[K] {
	return &counter[K]{
		metricDefs: metricDefs,
		counts:     make(map[string]map[[16]byte]*attrCounter, len(metricDefs)),
		timestamp:  time.Now(),
	}
}

type counter[K any] struct {
	metricDefs map[string]metricDef[K]
	counts     map[string]map[[16]byte]*attrCounter
	timestamp  time.Time
}

type attrCounter struct {
	attrs pcommon.Map
	count uint64
}

func (c *counter[K]) update(ctx context.Context, attrs pcommon.Map, tCtx K) error {
	var multiError error
	logger := log.New(os.Stdout, "", log.LstdFlags)
	for name, md := range c.metricDefs {
		countAttrs := pcommon.NewMap()
		for _, attr := range md.attrs {
			if attrVal, ok := attrs.Get(attr.Key); ok {
				logger.Println("************#@#########*********")
				logger.Println(attrVal)
				logger.Println(attrVal.Int())
				logger.Println(attrVal.Double())
				logger.Println(attrVal.Str())
				countAttrs.PutStr(attr.Key, fmt.Sprintf("%v", attrVal.Double()))
				//countAttrs.PutStr(attr.Key, attrVal.Str())
			} else if attr.DefaultValue != "" {
				countAttrs.PutStr(attr.Key, attr.DefaultValue)
			}
		}

		// Missing necessary attributes to be counted
		if countAttrs.Len() != len(md.attrs) {
			continue
		}

		// No conditions, so match all.
		if md.condition == nil {
			multiError = errors.Join(multiError, c.increment(name, countAttrs, 0))
			continue
		}

		if match, err := md.condition.Eval(ctx, tCtx); err != nil {
			multiError = errors.Join(multiError, err)
		} else if match {
			logger.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			logger.Println(countAttrs.AsRaw()) // pull value out of this map
			multiError = errors.Join(multiError, c.increment(name, countAttrs, 0))
		}
	}
	return multiError
}

func (c *counter[K]) increment(metricName string, attrs pcommon.Map, values int) error {
	if _, ok := c.counts[metricName]; !ok {
		c.counts[metricName] = make(map[[16]byte]*attrCounter)
	}

	key := noAttributes
	if attrs.Len() > 0 {
		key = pdatautil.MapHash(attrs)
	}

	if _, ok := c.counts[metricName][key]; !ok {
		c.counts[metricName][key] = &attrCounter{attrs: attrs}
	}
	logger := log.New(os.Stdout, "", log.LstdFlags)
	logger.Println("@#@#@@#@#@#@#@@#@#@#@#@#@#@#@#@#@#@")
	logger.Println(c)
	logger.Println(c.metricDefs)
	logger.Println(c.counts[metricName])
	logger.Println(c.counts[metricName][key])
	logger.Println(c.counts[metricName][key].attrs)
	logger.Println(c.counts[metricName][key].attrs.AsRaw())

	for strings, attr := range c.counts[metricName][key].attrs.AsRaw() {
		logger.Println(strings)
		logger.Println(attr)
		if attrVal, ok := c.counts[metricName][key].attrs.Get(strings); ok {
			logger.Println("*&*&*&*&*&*#@#@#@#@#@#*&*&*&*&*&@#@#@#")
			logger.Println(attrVal)
			logger.Println(attrVal.Int())
			logger.Println(attrVal.Double())
			logger.Println(attrVal.Str())
			val, _ := strconv.ParseUint(attrVal.Str(), 10, 64)
			logger.Println(val)
			c.counts[metricName][key].count = c.counts[metricName][key].count + val
	}
}

	return nil
}

func (c *counter[K]) appendMetricsTo(metricSlice pmetric.MetricSlice) {
	for name, md := range c.metricDefs {
		if len(c.counts[name]) == 0 {
			continue
		}
		countMetric := metricSlice.AppendEmpty()
		countMetric.SetName(name)
		countMetric.SetDescription(md.desc)
		sum := countMetric.SetEmptySum()
		// The delta value is always positive, so a value accumulated downstream is monotonic
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
		for _, dpCount := range c.counts[name] {
			dp := sum.DataPoints().AppendEmpty()
			dpCount.attrs.CopyTo(dp.Attributes())

			//This is where we do the numbers
			dp.SetIntValue(int64(dpCount.count))
			// TODO determine appropriate start time
			dp.SetTimestamp(pcommon.NewTimestampFromTime(c.timestamp))
		}
	}
}
