// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package span

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"

	m "github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/model/metadata"
	"github.com/elastic/apm-server/sourcemap"
	"github.com/elastic/apm-server/tests"
	"github.com/elastic/apm-server/transform"
)

func TestSpanTransform(t *testing.T) {
	path := "test/path"
	start := 0.65
	serviceName, serviceVersion, env := "myService", "1.2", "staging"
	service := metadata.Service{Name: serviceName, Version: serviceVersion, Environment: env}
	hexId, parentId, traceId := "0147258369012345", "abcdef0123456789", "01234567890123456789abcdefa"
	subtype := "amqp"
	action := "publish"
	timestamp := time.Date(2019, 1, 3, 15, 17, 4, 908.596*1e6,
		time.FixedZone("+0100", 3600))
	timestampUs := timestamp.UnixNano() / 1000
	method, statusCode, url := "get", 200, "http://localhost"
	instance, statement, dbType, user, rowsAffected := "db01", "select *", "sql", "jane", 5
	metadataLabels := common.MapStr{"label.a": "a", "label.b": "b", "c": 1}
	metadata := metadata.Metadata{Service: service, Labels: metadataLabels}
	address, port := "127.0.0.1", 8080
	destServiceType, destServiceName, destServiceResource := "db", "elasticsearch", "elasticsearch"

	tests := []struct {
		Event  Event
		Output common.MapStr
		Msg    string
	}{
		{
			Event: Event{Timestamp: timestamp, Metadata: metadata},
			Output: common.MapStr{
				"processor": common.MapStr{"event": "span", "name": "transaction"},
				"service":   common.MapStr{"name": serviceName, "environment": env, "version": serviceVersion},
				"span": common.MapStr{
					"duration": common.MapStr{"us": 0},
					"name":     "",
					"type":     "",
				},
				"labels":    metadataLabels,
				"timestamp": common.MapStr{"us": timestampUs},
			},
			Msg: "Span without a Stacktrace",
		},
		{
			Event: Event{
				Metadata:   metadata,
				ID:         hexId,
				TraceID:    &traceId,
				ParentID:   &parentId,
				Name:       "myspan",
				Type:       "myspantype",
				Subtype:    &subtype,
				Action:     &action,
				Timestamp:  timestamp,
				Start:      &start,
				Duration:   1.20,
				Stacktrace: m.Stacktrace{{AbsPath: &path}},
				Labels:     common.MapStr{"label.a": 12},
				HTTP:       &HTTP{Method: &method, StatusCode: &statusCode, URL: &url},
				DB: &DB{
					Instance:     &instance,
					Statement:    &statement,
					Type:         &dbType,
					UserName:     &user,
					RowsAffected: &rowsAffected},
				Destination: &Destination{Address: &address, Port: &port},
				DestinationService: &DestinationService{
					Type:     &destServiceType,
					Name:     &destServiceName,
					Resource: &destServiceResource,
				},
				Message: &m.Message{QueueName: tests.StringPtr("users")},
			},
			Output: common.MapStr{
				"span": common.MapStr{
					"id":       hexId,
					"duration": common.MapStr{"us": 1200},
					"name":     "myspan",
					"start":    common.MapStr{"us": 650},
					"type":     "myspantype",
					"subtype":  subtype,
					"action":   action,
					"stacktrace": []common.MapStr{{
						"exclude_from_grouping": false,
						"abs_path":              path,
						"sourcemap": common.MapStr{
							"error":   "Colno mandatory for sourcemapping.",
							"updated": false,
						}}},
					"db": common.MapStr{
						"instance":      instance,
						"statement":     statement,
						"type":          dbType,
						"user":          common.MapStr{"name": user},
						"rows_affected": rowsAffected,
					},
					"http": common.MapStr{
						"url":      common.MapStr{"original": url},
						"response": common.MapStr{"status_code": statusCode},
						"method":   "get",
					},
					"destination": common.MapStr{
						"service": common.MapStr{
							"type":     destServiceType,
							"name":     destServiceName,
							"resource": destServiceResource,
						},
					},
					"message": common.MapStr{"queue": common.MapStr{"name": "users"}},
				},
				"labels":      common.MapStr{"label.a": 12, "label.b": "b", "c": 1},
				"processor":   common.MapStr{"event": "span", "name": "transaction"},
				"service":     common.MapStr{"name": serviceName, "environment": env, "version": serviceVersion},
				"timestamp":   common.MapStr{"us": timestampUs},
				"trace":       common.MapStr{"id": traceId},
				"parent":      common.MapStr{"id": parentId},
				"destination": common.MapStr{"address": address, "ip": address, "port": port},
			},
			Msg: "Full Span",
		},
	}

	tctx := &transform.Context{
		Config: transform.Config{SourcemapStore: &sourcemap.Store{}},
	}
	for _, test := range tests {
		output := test.Event.Transform(context.Background(), tctx)
		fields := output[0].Fields
		assert.Equal(t, test.Output, fields)
	}
}
