// Copyright 2019 The OpenSDS Authors.
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

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/micro/go-micro/metadata"
	"github.com/opensds/multi-cloud/api/pkg/common"
	. "github.com/opensds/multi-cloud/datamover/pkg/utils"
	"github.com/opensds/multi-cloud/datamover/proto"
	"github.com/opensds/multi-cloud/s3/pkg/utils"
	osdss3 "github.com/opensds/multi-cloud/s3/proto"
	log "github.com/sirupsen/logrus"
)

func loadStorageClassDefinition() error {
	res, _ := s3client.GetTierMap(context.Background(), &osdss3.BaseRequest{})
	if len(res.Tier2Name) == 0 {
		log.Info("get tier definition failed")
		return fmt.Errorf("get tier definition failed")
	}

	log.Infof("Load storage class definition from s3 service successfully, res.Tier2Name:%+v\n", res.Tier2Name)
	Int2ExtTierMap = make(map[string]*Int2String)
	for k, v := range res.Tier2Name {
		val := make(Int2String)
		for k1, v1 := range v.Lst {
			val[k1] = v1
		}
		Int2ExtTierMap[k] = &val
	}

	return nil
}

func doInCloudTransition(acReq *datamover.LifecycleActionRequest) error {
	log.Infof("in-cloud transition action: transition %s from %d to %d of %s.\n",
		acReq.ObjKey, acReq.SourceTier, acReq.TargetTier, acReq.SourceBackend)

	log.Infof("in-cloud transition of object[%s], bucket:%v, target tier:%d\n", acReq.ObjKey, acReq.BucketName,
		acReq.TargetTier)
	req := &osdss3.MoveObjectRequest{
		SrcObject:  acReq.ObjKey,
		SrcBucket:  acReq.BucketName,
		TargetTier: acReq.TargetTier,
		CopyType:   utils.MoveType_ChangeStorageTier,
		SourceType: utils.CopySourceType_Lifecycle,
	}

	// add object to InProgressObjs
	if _, ok := InProgressObjs[acReq.ObjKey]; !ok {
		InProgressObjs[acReq.ObjKey] = struct{}{}
	} else {
		log.Infof("the transition of object[%s] is in-progress\n", acReq.ObjKey)
		return errors.New(DMERR_TransitionInprogress)
	}

	ctx, _ := context.WithTimeout(context.Background(), CLOUD_OPR_TIMEOUT*time.Second)
	ctx = metadata.NewContext(ctx, map[string]string{
		common.CTX_KEY_IS_ADMIN:  strconv.FormatBool(true),
		common.CTX_KEY_TENANT_ID: INTERNAL_TENANT,
	})
	_, err := s3client.MoveObject(ctx, req)
	if err != nil {
		// if failed, it will try again in the next round schedule
		log.Errorf("in-cloud transition of %s failed:%v\n", acReq.ObjKey, err)
	} else {
		log.Infof("in-cloud transition of %s succeed.\n", acReq.ObjKey)
	}

	// remove object from InProgressObjs
	delete(InProgressObjs, acReq.ObjKey)

	return err
}
