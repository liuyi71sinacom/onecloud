// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/options"
)

const (
	MeterAlertMetadataType      = "type"
	MeterAlertMetadataProjectId = "project_id"
	MeterAlertMetadataAccountId = "account_id"
	MeterAlertMetadataProvider  = "provider"
)

var MeterAlertManager *SMeterAlertManager

func init() {
	MeterAlertManager = NewMeterAlertManager()
}

type IMeterAlertDriver interface {
	GetType() string
	GetName() string
	ToAlertCreateInput(input monitor.MeterAlertCreateInput, notificatoins []string, allAccountIds []string) monitor.AlertCreateInput
}

type SMeterAlertManager struct {
	SV1AlertManager

	drivers map[string]IMeterAlertDriver
}

func NewMeterAlertManager() *SMeterAlertManager {
	man := &SMeterAlertManager{
		SV1AlertManager: SV1AlertManager{
			*NewAlertManager(SMeterAlert{}, "meteralert", "meteralerts"),
		},
	}
	man.SetVirtualObject(man)
	man.registerDriver(man.newDailyFeeDriver())
	man.registerDriver(man.newMonthFeeDriver())
	return man
}

type SMeterAlert struct {
	SV1Alert
}

func (man *SMeterAlertManager) newDailyFeeDriver() IMeterAlertDriver {
	return new(sMeterDailyFee)
}

func (man *SMeterAlertManager) newMonthFeeDriver() IMeterAlertDriver {
	return new(sMeterMonthFee)
}

func (man *SMeterAlertManager) registerDriver(drv IMeterAlertDriver) {
	if man.drivers == nil {
		man.drivers = make(map[string]IMeterAlertDriver, 0)
	}
	man.drivers[drv.GetType()] = drv
}

func (man *SMeterAlertManager) GetDriver(typ string) IMeterAlertDriver {
	return man.drivers[typ]
}

func (man *SMeterAlertManager) genName(ownerId mcclient.IIdentityProvider, hint string) (string, error) {
	return db.GenerateName(man, ownerId, hint)
}

func (man *SMeterAlertManager) getAllBillAccounts(ctx context.Context) ([]jsonutils.JSONObject, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	q := jsonutils.NewDict()
	q.Add(jsonutils.NewString("accountList"), "account_id")
	q.Add(jsonutils.NewInt(-1), "limit")
	ret, err := modules.BillBalances.List(s, q)
	if err != nil {
		return nil, err
	}
	return ret.Data, nil
}

func (man *SMeterAlertManager) getAllBillAccountIds(ctx context.Context) ([]string, error) {
	objs, err := man.getAllBillAccounts(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(objs))
	for idx, obj := range objs {
		id, err := obj.GetString("id")
		if err != nil {
			return nil, err
		}
		ids[idx] = id
	}
	return ids, nil
}

func (man *SMeterAlertManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.MeterAlertCreateInput) (*monitor.MeterAlertCreateInput, error) {
	if data.Period == "" {
		// default 30 minutes
		data.Period = "30m"
	}
	if data.Window == "" {
		// default 5 minutes
		data.Window = "5m"
	}
	if _, err := time.ParseDuration(data.Period); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid period format: %s", data.Period)
	}
	if data.Recipients == "" {
		return nil, httperrors.NewInputParameterError("recipients is empty")
	}
	notification, err := man.CreateNotification(ctx, userCred, data.Type, data.Channel, data.Recipients)
	if err != nil {
		return nil, errors.Wrap(err, "create notification")
	}

	if data.ProjectId == "" {
		return nil, httperrors.NewInputParameterError("project_id is empty")
	}

	drv := man.GetDriver(data.Type)
	if drv == nil {
		return nil, httperrors.NewInputParameterError("not support type %q", data.Type)
	}
	name, err := man.genName(ownerId, drv.GetName())
	if err != nil {
		return nil, err
	}
	allAccountIds := []string{}
	if data.AccountId == "" {
		allAccountIds, err = man.getAllBillAccountIds(ctx)
		if err != nil {
			return nil, err
		}
	}
	alertInput := drv.ToAlertCreateInput(
		data, []string{notification.GetId()},
		allAccountIds)
	alertInput, err = AlertManager.ValidateCreateData(ctx, userCred, ownerId, query, alertInput)
	if err != nil {
		return nil, err
	}
	data.Name = name
	data.AlertCreateInput = &alertInput
	return &data, nil
}

type sMeterDailyFee struct{}

func (_ *sMeterDailyFee) GetType() string {
	return monitor.MeterAlertTypeDailyResFee
}

func (_ *sMeterDailyFee) GetName() string {
	return "日消费"
}

func (f *sMeterDailyFee) ToAlertCreateInput(
	input monitor.MeterAlertCreateInput,
	notifications []string,
	allAccountIds []string,
) monitor.AlertCreateInput {
	freq, _ := time.ParseDuration(input.Window)
	ret := monitor.AlertCreateInput{
		Name:      f.GetName(),
		Frequency: int64(freq / time.Second),
		Settings: GetMeterAlertSetting(input, notifications,
			"account_daily_resfee",
			"meter_db", allAccountIds, "sumDate"),
	}
	return ret
}

type sMeterMonthFee struct{}

func (_ *sMeterMonthFee) GetType() string {
	return monitor.MeterAlertTypeMonthResFee
}

func (_ *sMeterMonthFee) GetName() string {
	return "月消费"
}

func (f *sMeterMonthFee) ToAlertCreateInput(
	input monitor.MeterAlertCreateInput,
	notifications []string,
	allAccountIds []string,
) monitor.AlertCreateInput {
	freq, _ := time.ParseDuration(input.Window)
	ret := monitor.AlertCreateInput{
		Name:      f.GetName(),
		Frequency: int64(freq / time.Second),
		Settings: GetMeterAlertSetting(input, notifications,
			"account_month_resfee",
			"meter_db", allAccountIds, "sumMonth"),
	}
	return ret
}

func GetMeterAlertSetting(
	input monitor.MeterAlertCreateInput,
	ns []string,
	measurement string,
	db string,
	accountIds []string,
	groupByStr string,
) monitor.AlertSetting {
	q, reducer, eval := GetMeterAlertQuery(input, measurement, db, accountIds, groupByStr)
	return monitor.AlertSetting{
		Level:         input.Level,
		Notifications: ns,
		Conditions: []monitor.AlertCondition{
			{
				Type:     "query",
				Operator: "and",
				Query: monitor.AlertQuery{
					Model: q,
					From:  input.Period,
					To:    "now",
				},
				Reducer:   reducer,
				Evaluator: eval,
			},
		},
	}
}

func GetMeterAlertQuery(
	input monitor.MeterAlertCreateInput,
	measurement string,
	db string,
	allAccountIds []string,
	groupByStr string,
) (
	monitor.MetricQuery,
	monitor.Condition,
	monitor.Condition) {
	var (
		evaluator, reducer monitor.Condition
		alertType, field   string
		filters            []monitor.MetricQueryTag
	)
	groupBy := []monitor.MetricQueryPart{}
	evaluator = monitor.GetNodeAlertEvaluator(input.Comparator, input.Threshold)

	if input.AccountId == "" {
		reducer = monitor.Condition{Type: "sum"}
		alertType = "overview"
		field = "sum"
		for _, aId := range allAccountIds {
			filters = append(filters, monitor.MetricQueryTag{
				Key:       "accountId",
				Value:     aId,
				Condition: "or",
			})
		}
	} else {
		reducer = monitor.Condition{Type: "avg"}
		alertType = "account"
		field = input.Type
		groupBy = append(groupBy, monitor.MetricQueryPart{
			Type:   "field",
			Params: []string{field},
		})
		filters = append(filters, monitor.MetricQueryTag{
			Key:       "accountId",
			Value:     input.AccountId,
			Condition: "and",
		})
		filters = append(filters, monitor.MetricQueryTag{
			Key:   "provider",
			Value: input.Provider,
		})
	}

	log.Debugf("==alertType: %s", alertType)

	if input.ProjectId != "" {
		filters = append(filters, monitor.MetricQueryTag{
			Key:   "projectId",
			Value: input.ProjectId,
		})
	}

	groupBy = append(groupBy, monitor.MetricQueryPart{
		Type:   "field",
		Params: []string{groupByStr},
	})

	sels := make([]monitor.MetricQuerySelect, 0)
	sels = append(sels, monitor.NewMetricQuerySelect(
		monitor.MetricQueryPart{
			Type:   "field",
			Params: []string{input.Type},
		}))
	q := monitor.MetricQuery{
		Selects:     sels,
		Tags:        filters,
		GroupBy:     groupBy,
		Measurement: measurement,
		Database:    db,
	}
	return q, reducer, evaluator
}

func (man *SMeterAlertManager) GetAlert(id string) (*SMeterAlert, error) {
	obj, err := man.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SMeterAlert), nil
}

func (man *SMeterAlertManager) CustomizeFilterList(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, query jsonutils.JSONObject) (
	*db.CustomizeListFilters, error) {
	filters, err := man.SV1AlertManager.CustomizeFilterList(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	input := new(monitor.MeterAlertListInput)
	if err := query.Unmarshal(input); err != nil {
		return nil, err
	}
	wrapF := func(f func(obj *SMeterAlert) (bool, error)) func(object jsonutils.JSONObject) (bool, error) {
		return func(data jsonutils.JSONObject) (bool, error) {
			id, err := data.GetString("id")
			if err != nil {
				return false, err
			}
			obj, err := man.GetAlert(id)
			if err != nil {
				return false, err
			}
			return f(obj)
		}
	}

	if input.Type != "" {
		filters.Append(wrapF(func(obj *SMeterAlert) (bool, error) {
			return obj.getType() == input.Type, nil
		}))
	}

	if input.AccountId != "" {
		filters.Append(wrapF(func(obj *SMeterAlert) (bool, error) {
			return obj.getAccountId() == input.AccountId, nil
		}))
	}

	if input.Provider != "" {
		filters.Append(wrapF(func(obj *SMeterAlert) (bool, error) {
			return obj.getProvider() == input.Provider, nil
		}))
	}

	if input.ProjectId != "" {
		filters.Append(wrapF(func(obj *SMeterAlert) (bool, error) {
			return obj.getProjectId() == input.ProjectId, nil
		}))
	}

	return filters, nil
}

func (alert *SMeterAlert) setType(ctx context.Context, userCred mcclient.TokenCredential, t string) error {
	return alert.SetMetadata(ctx, MeterAlertMetadataType, t, userCred)
}

func (alert *SMeterAlert) getType() string {
	return alert.GetMetadata(MeterAlertMetadataType, nil)
}

func (alert *SMeterAlert) setProjectId(ctx context.Context, userCred mcclient.TokenCredential, id string) error {
	return alert.SetMetadata(ctx, MeterAlertMetadataProjectId, id, userCred)
}

func (alert *SMeterAlert) getProjectId() string {
	return alert.GetMetadata(MeterAlertMetadataProjectId, nil)
}

func (alert *SMeterAlert) setAccountId(ctx context.Context, userCred mcclient.TokenCredential, id string) error {
	return alert.SetMetadata(ctx, MeterAlertMetadataAccountId, id, userCred)
}

func (alert *SMeterAlert) getAccountId() string {
	return alert.GetMetadata(MeterAlertMetadataAccountId, nil)
}

func (alert *SMeterAlert) setProvider(ctx context.Context, userCred mcclient.TokenCredential, p string) error {
	return alert.SetMetadata(ctx, MeterAlertMetadataProvider, p, userCred)
}

func (alert *SMeterAlert) getProvider() string {
	return alert.GetMetadata(MeterAlertMetadataProvider, nil)
}

func (alert *SMeterAlert) PostCreate(ctx context.Context,
	userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	alert.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := new(monitor.MeterAlertCreateInput)
	if err := data.Unmarshal(input); err != nil {
		log.Errorf("post create unmarshal input: %v", err)
		return
	}
	if input.Type != "" {
		if err := alert.setType(ctx, userCred, input.Type); err != nil {
			log.Errorf("set type: %v", err)
		}
	}
	if input.Provider != "" {
		if err := alert.setProvider(ctx, userCred, input.Provider); err != nil {
			log.Errorf("set proider: %v", err)
		}
	}
	if input.AccountId != "" {
		if err := alert.setAccountId(ctx, userCred, input.AccountId); err != nil {
			log.Errorf("set account_id: %v", err)
		}
	}
	if input.ProjectId != "" {
		if err := alert.setProjectId(ctx, userCred, input.ProjectId); err != nil {
			log.Errorf("set project_id: %v", err)
		}
	}
}

func (alert *SMeterAlert) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (monitor.MeterAlertDetails, error) {
	var err error
	out := monitor.MeterAlertDetails{}
	commonDetails, err := alert.SV1Alert.GetExtraDetails(ctx, userCred, query, isList)
	if err != nil {
		return out, err
	}
	out.AlertV1Details = commonDetails

	out.Type = alert.getType()
	out.ProjectId = alert.getProjectId()
	out.Provider = alert.getProvider()
	out.AccountId = alert.getAccountId()

	return out, nil
}