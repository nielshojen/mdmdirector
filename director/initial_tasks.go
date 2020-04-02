package director

import (
	"time"

	"github.com/mdmdirector/mdmdirector/utils"

	"github.com/mdmdirector/mdmdirector/db"
	"github.com/mdmdirector/mdmdirector/log"
	"github.com/mdmdirector/mdmdirector/types"
	"github.com/pkg/errors"
)

func RunInitialTasks(udid string) error {
	if udid == "" {
		err := errors.New("No Device UDID")
		return errors.Wrap(err, "RunInitialTasks")
	}

	device, err := GetDevice(udid)
	if err != nil {
		return errors.Wrap(err, "RunInitialTasks")
	}

	log.Info("Running initial tasks")

	if utils.ResetDeviceProfilesAtEnrollment() {
		err = ResetDeviceProfiles(device)
		if err != nil {
			return errors.Wrap(err, "RunInitialTasks")
		}
	}
	err = ClearCommands(&device)
	if err != nil {
		return err
	}

	_, err = InstallAllProfiles(device)
	if err != nil {
		return errors.Wrap(err, "RunInitialTasks:InstallAllProfiles")
	}

	_, err = InstallBootstrapPackages(device)
	if err != nil {
		return errors.Wrap(err, "RunInitialTasks:InstallBootstrapPackages")
	}

	// commandsList := append(profileCommands, packageCommands...)
	// var uuidList []string
	// for _, command := range commandsList {
	// 	uuidList = append(uuidList, command.CommandUUID)
	// }

	err = processDeviceConfigured(device)
	if err != nil {
		return errors.Wrap(err, "RunInitialTasks:processDeviceConfigured")
	}

	return nil
}

func processDeviceConfigured(device types.Device) error {
	var deviceModel types.Device
	var err error
	err = SendDeviceConfigured(device)
	if err != nil {
		return errors.Wrap(err, "RunInitialTasks")
	}
	err = SaveDeviceConfigured(device)
	if err != nil {
		return err
	}
	err = db.DB.Model(&deviceModel).Where("ud_id = ?", device.UDID).Update(map[string]interface{}{"last_info_requested": time.Now()}).Error
	if err != nil {
		return err
	}

	// RequestSecurityInfo(device)
	// RequestDeviceInformation(device)
	// RequestProfileList(device)
	return nil
}

func SendDeviceConfigured(device types.Device) error {
	requestType := "DeviceConfigured"
	var commandPayload types.CommandPayload
	commandPayload.UDID = device.UDID
	commandPayload.RequestType = requestType
	_, err := SendCommand(commandPayload)
	if err != nil {
		return errors.Wrap(err, "SendDeviceConfigured")
	}
	// Twice for luck
	_, err = SendCommand(commandPayload)
	if err != nil {
		return errors.Wrap(err, "SendDeviceConfigured")
	}
	return nil
}

func SaveDeviceConfigured(device types.Device) error {
	var deviceModel types.Device
	// err := db.DB.Model(&deviceModel).Where("ud_id = ?", device.UDID).Update(map[string]interface{}{"awaiting_configuration": false, "token_update_received": true, "authenticate_received": true, "initial_tasks_run": true}).Error
	err := db.DB.Model(&deviceModel).Where("ud_id = ?", device.UDID).Update(map[string]interface{}{"token_update_received": true, "authenticate_received": true, "initial_tasks_run": true}).Error
	if err != nil {
		return err
	}

	return nil
}

func ResetDevice(device types.Device) error {
	var deviceModel types.Device
	err := ClearCommands(&device)
	if err != nil {
		return errors.Wrap(err, "ResetDevice:ClearCommands")
	}
	log.Infof("Resetting %v", device.UDID)
	err = db.DB.Model(&deviceModel).Where("ud_id = ?", device.UDID).Update(map[string]interface{}{"token_update_received": false, "authenticate_received": false, "initial_tasks_run": false}).Error
	if err != nil {
		return errors.Wrap(err, "reset device")
	}

	return nil
}
