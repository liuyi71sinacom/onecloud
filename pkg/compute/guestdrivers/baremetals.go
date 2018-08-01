package guestdrivers

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db/quotas"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type SBaremetalGuestDriver struct {
	SBaseGuestDriver
}

func init() {
	driver := SBaremetalGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SBaremetalGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_BAREMETAL
}

func (self *SBaremetalGuestDriver) GetMaxVCpuCount() int {
	return 1024
}

func (self *SBaremetalGuestDriver) GetMaxVMemSizeGB() int {
	return 4096
}

func (self *SBaremetalGuestDriver) PrepareDiskRaidConfig(host *models.SHost, params *jsonutils.JSONDict) error {
	/*confs := params.GetArray()
	baremetal.CalculateLayout()
	from clouds.baremetal import diskconfig
	confs = task.params.get('baremetal_disk_config', None)
	if confs is None:
	confs = [diskconfig.parse_diskconfig('')]
	layouts = diskconfig.calculate_layout(confs, baremetal.storage_info)
	baremetal.update_disk_config(layouts)*/

	// TODO

	return nil
}

func (self *SBaremetalGuestDriver) GetNamedNetworkConfiguration(guest *models.SGuest, userCred mcclient.TokenCredential, host *models.SHost, netConfig *models.SNetworkConfig) (*models.SNetwork, string, int8, models.IPAddlocationDirection) {
	net, _ := host.GetNetworkWithIdAndCredential(netConfig.Network, userCred, netConfig.Reserved)
	return net, netConfig.Mac, -1, models.IPAllocationStepdown
}

func (self *SBaremetalGuestDriver) Attach2RandomNetwork(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *models.SNetworkConfig, pendingUsage quotas.IQuota) error {
	// TODO

	return nil
}

func (self *SBaremetalGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	bs := host.GetBaremetalstorage()
	return bs.GetStorage()
}

func (self *SBaremetalGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	task.ScheduleRun(nil) // skip
	return nil
}

func (self *SBaremetalGuestDriver) RequestGuestCreateInsertIso(ctx context.Context, imageId string, guest *models.SGuest, task taskman.ITask) error {
	task.ScheduleRun(nil) // skip
	return nil
}

func (self *SBaremetalGuestDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error) {
	data := jsonutils.NewDict()
	// TODO
	return data, nil
}

func (self *SBaremetalGuestDriver) RequestStopGuestForDelete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// TODO
	return nil
}

func (self *SBaremetalGuestDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// TODO
	return nil
}

func (self *SBaremetalGuestDriver) StartGuestStopTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	// TODO
	return nil
}

func (self *SBaremetalGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// TODO
	return nil
}

func (self *SBaremetalGuestDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	data := jsonutils.NewDict()
	// TODO
	return data, nil
}

func (self *SBaremetalGuestDriver) StartGuestSyncstatusTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	// TODO
	return nil
}

func (self *SBaremetalGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SBaremetalGuestDriver) ValidateCreateHostData(ctx context.Context, userCred mcclient.TokenCredential, bmName string, host *models.SHost, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SBaremetalGuestDriver) GetJsonDescAtHost(ctx context.Context, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	return guest.GetJsonDescAtBaremetal(ctx, host)
}

func (self *SBaremetalGuestDriver) GetGuestVncInfo(userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(host.Id), "host_id")
	zone := host.GetZone()
	data.Add(jsonutils.NewString(zone.Name), "zone")
	return data, nil
}

func (self *SBaremetalGuestDriver) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, data *jsonutils.JSONDict) error {
	return guest.StartGueststartTask(ctx, userCred, data, "")
}

func (self *SBaremetalGuestDriver) CheckDiskTemplateOnStorage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, storageId string, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaremetalGuestDriver) OnGuestDeployTaskComplete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// TODO
	return nil
}

func (self *SBaremetalGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {
	// TODO
	return nil
}

func (self *SBaremetalGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// TODO

	return nil
}

func (self *SBaremetalGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// TODO

	return nil
}
