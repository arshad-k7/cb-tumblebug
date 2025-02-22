/*
Copyright 2019 The Cloud-Barista Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package mcis is to manage multi-cloud infra service
package mcis

import (
	"encoding/json"
	"fmt"

	"github.com/cloud-barista/cb-tumblebug/src/core/common"
	"github.com/cloud-barista/cb-tumblebug/src/core/mcir"
	"github.com/go-resty/resty/v2"
)

type TbVmSnapshotReq struct {
	Name string `json:"name" example:"aws-ap-southeast-1-snapshot"`
}

// CreateVmSnapshot is func to create VM snapshot
func CreateVmSnapshot(nsId string, mcisId string, vmId string, snapshotName string) (mcir.TbCustomImageInfo, error) {
	vmKey := common.GenMcisKey(nsId, mcisId, vmId)

	// Check existence of the key. If no key, no update.
	keyValue, err := common.CBStore.Get(vmKey)
	if keyValue == nil || err != nil {
		err := fmt.Errorf("Failed to find 'ns/mcis/vm': %s/%s/%s \n", nsId, mcisId, vmId)
		common.CBLog.Error(err)
		return mcir.TbCustomImageInfo{}, err
	}

	vm := TbVmInfo{}
	json.Unmarshal([]byte(keyValue.Value), &vm)

	if snapshotName == "" {
		snapshotName = fmt.Sprintf("%s-%s", vm.Name, common.GenerateNewRandomString(5))
	}

	tempReq := mcir.SpiderMyImageReq{
		ConnectionName: vm.ConnectionName,
		ReqInfo: struct {
			Name     string
			SourceVM string
		}{
			Name:     snapshotName,
			SourceVM: vm.CspViewVmDetail.IId.NameId,
		},
	}

	client := resty.New().SetCloseConnection(true)
	client.SetAllowGetMethodPayload(true)

	req := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(tempReq).
		SetResult(&mcir.SpiderMyImageInfo{}) // or SetResult(AuthSuccess{}).
		//SetError(&AuthError{}).       // or SetError(AuthError{}).

	// Inspect DataDisks before creating VM snapshot
	// Disabled because: there is no difference in dataDisks before and after creating VM snapshot
	// inspect_result_before_snapshot, err := InspectResources(vm.ConnectionName, common.StrDataDisk)
	// dataDisks_before_snapshot := inspect_result_before_snapshot.Resources.OnTumblebug.Info
	// if err != nil {
	// 	err := fmt.Errorf("Failed to get current datadisks' info. \n")
	// 	common.CBLog.Error(err)
	// 	return mcir.TbCustomImageInfo{}, err
	// }

	// Create VM snapshot
	url := fmt.Sprintf("%s/myimage", common.SpiderRestUrl)

	resp, err := req.Post(url)

	fmt.Printf("HTTP Status code: %d \n", resp.StatusCode())
	switch {
	case resp.StatusCode() >= 400 || resp.StatusCode() < 200:
		err := fmt.Errorf(string(resp.Body()))
		fmt.Println("body: ", string(resp.Body()))
		common.CBLog.Error(err)
		return mcir.TbCustomImageInfo{}, err
	}

	// create one customImage
	tempSpiderMyImageInfo := resp.Result().(*mcir.SpiderMyImageInfo)
	tempTbCustomImageInfo := mcir.TbCustomImageInfo{
		Namespace:            nsId,
		Id:                   "", // This field will be assigned in RegisterCustomImageWithInfo()
		Name:                 snapshotName,
		ConnectionName:       vm.ConnectionName,
		SourceVmId:           vmId,
		CspCustomImageId:     tempSpiderMyImageInfo.IId.SystemId,
		CspCustomImageName:   tempSpiderMyImageInfo.IId.NameId,
		Description:          "",
		CreationDate:         tempSpiderMyImageInfo.CreatedTime,
		GuestOS:              "",
		Status:               tempSpiderMyImageInfo.Status,
		KeyValueList:         tempSpiderMyImageInfo.KeyValueList,
		AssociatedObjectList: []string{},
		IsAutoGenerated:      false,
		SystemLabel:          "",
	}

	result, err := mcir.RegisterCustomImageWithInfo(nsId, tempTbCustomImageInfo)
	if err != nil {
		err := fmt.Errorf("Failed to find 'ns/mcis/vm': %s/%s/%s \n", nsId, mcisId, vmId)
		common.CBLog.Error(err)
		return mcir.TbCustomImageInfo{}, err
	}

	// Inspect DataDisks after creating VM snapshot
	// Disabled because: there is no difference in dataDisks before and after creating VM snapshot
	// inspect_result_after_snapshot, err := InspectResources(vm.ConnectionName, common.StrDataDisk)
	// dataDisks_after_snapshot := inspect_result_after_snapshot.Resources.OnTumblebug.Info
	// if err != nil {
	// 	err := fmt.Errorf("Failed to get current datadisks' info. \n")
	// 	common.CBLog.Error(err)
	// 	return mcir.TbCustomImageInfo{}, err
	// }

	// difference_dataDisks := Difference_dataDisks(dataDisks_before_snapshot, dataDisks_after_snapshot)

	// // create 'n' dataDisks
	// for _, v := range difference_dataDisks {
	// 	tempTbDataDiskReq := mcir.TbDataDiskReq{
	// 		Name:           fmt.Sprintf("%s-%s", vm.Name, common.GenerateNewRandomString(5)),
	// 		ConnectionName: vm.ConnectionName,
	// 		CspDataDiskId:  v.IdByCsp,
	// 	}

	// 	_, err = mcir.CreateDataDisk(nsId, &tempTbDataDiskReq, "register")
	// 	if err != nil {
	// 		err := fmt.Errorf("Failed to register the created dataDisk %s to TB. \n", v.IdByCsp)
	// 		common.CBLog.Error(err)
	// 		continue
	// 	}
	// }

	return result, nil
}

func Difference_dataDisks(a, b []resourceOnTumblebugInfo) []resourceOnTumblebugInfo {
	mb := make(map[interface{}]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []resourceOnTumblebugInfo
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
