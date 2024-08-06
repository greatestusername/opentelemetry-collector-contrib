// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sumconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/sumconnector"

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil"
)

var noAttributes = [16]byte{}

func newSummer[K any](metricDefs map[string]metricDef[K]) *summer[K] {
	return &summer[K]{
		metricDefs: metricDefs,
		sums:       make(map[string]map[[16]byte]*attrSummer, len(metricDefs)),
		timestamp:  time.Now(),
	}
}

type summer[K any] struct {
	metricDefs map[string]metricDef[K]
	sums       map[string]map[[16]byte]*attrSummer
	timestamp  time.Time
}

type attrSummer struct {
	attrs pcommon.Map
	sum   float64
}

func (c *summer[K]) update(ctx context.Context, attrs pcommon.Map, tCtx K) error {
	var multiError error
	for name, md := range c.metricDefs {
		sourceAttribute := md.sourceAttr
		sumAttrs := pcommon.NewMap()
		for _, attr := range md.attrs {
			if attrVal, ok := attrs.Get(attr.Key); ok {
				fmt.Println("FIRST!!! attrVal is: " + attrVal.AsString())
				switch {
				case attrVal.Str() != "":
					sumAttrs.PutStr(attr.Key, attrVal.Str())
				case attrVal.Double() != 0:
					sumAttrs.PutStr(attr.Key, fmt.Sprintf("%v", attrVal.Double()))
				case attrVal.Int() != 0:
					sumAttrs.PutStr(attr.Key, fmt.Sprintf("%v", attrVal.Int()))
				}
				}
			}
			
			// Missing necessary attributes
			if sumAttrs.Len() != len(md.attrs) {
				continue
			}
			
			if sourceAttrVal, ok := attrs.Get(sourceAttribute); ok {
				sumAttrs.PutStr(sourceAttribute, sourceAttrVal.AsString())
				fmt.Println("Added sourceAttribute!!!!!")
			}

		// No conditions, so match all.
		if md.condition == nil {
			multiError = errors.Join(multiError, c.increment(name, sourceAttribute, sumAttrs))
			continue
		}

		if match, err := md.condition.Eval(ctx, tCtx); err != nil {
			multiError = errors.Join(multiError, err)
		} else if match {
			multiError = errors.Join(multiError, c.increment(name, sourceAttribute, sumAttrs))
		}
	}
	return multiError
}

func (c *summer[K]) increment(metricName string, sourceAttribute string, attrs pcommon.Map) error {
	if _, ok := c.sums[metricName]; !ok {
		c.sums[metricName] = make(map[[16]byte]*attrSummer)
	}

	key := noAttributes
	if attrs.Len() > 0 {
		key = pdatautil.MapHash(attrs)
	}
	
	if _, ok := c.sums[metricName][key]; !ok {
		c.sums[metricName][key] = &attrSummer{attrs: attrs}
	}
	
	for i := range c.sums[metricName][key].attrs.AsRaw() {
		fmt.Println("Value of i is: " + i)
		if i == sourceAttribute {
			if attrVal, ok := c.sums[metricName][key].attrs.Get(sourceAttribute); ok {
				fmt.Println("Value of attrVal is: " + attrVal.AsString())
				val, _ := strconv.ParseFloat(attrVal.Str(), 64)
				fmt.Println(val)
				c.sums[metricName][key].sum += val
				fmt.Println(c.sums[metricName][key].sum)
			}
		}
	}

	return nil
}

func (c *summer[K]) appendMetricsTo(metricSlice pmetric.MetricSlice) {
	for name, md := range c.metricDefs {
		if len(c.sums[name]) == 0 {
			continue
		}
		sumMetric := metricSlice.AppendEmpty()
		sumMetric.SetName(name)
		sumMetric.SetDescription(md.desc)
		sum := sumMetric.SetEmptySum()
		// The delta value is always positive, so a value accumulated downstream is monotonic
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
		i := 0
		for _, dpCount := range c.sums[name] {
			dp := sum.DataPoints().AppendEmpty()
			dpCount.attrs.CopyTo(dp.Attributes())
			fmt.Println(dpCount.attrs.AsRaw())
			dp.SetDoubleValue(float64(dpCount.sum))
			// TODO determine appropriate start time
			dp.SetTimestamp(pcommon.NewTimestampFromTime(c.timestamp))
			i += 1
			fmt.Println(i)
		}
	}
}
