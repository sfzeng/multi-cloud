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

package utils

import (
	"crypto/md5"
	"encoding/hex"
)

type Database struct {
	Credential string `conf:"credential,username:password@tcp(ip:port)/dbname"`
	Driver     string `conf:"driver,mongodb"`
	Endpoint   string `conf:"endpoint,localhost:27017"`
}

// Tier1, Tier99 and Tier999 just like the tiers of hot, warm, cold.
// In the future, we will provide the ability for users to add new storage tiers, if we use 1, 2 and 3, then no space for new storage tiers.
const (
	Tier1   = 1
	Tier99  = 99
	Tier999 = 999
)

const (
	AWS_STANDARD    = "STANDARD"
	AWS_STANDARD_IA = "STANDARD_IA"
	AWS_GLACIER     = "GLACIER"
)

const (
	CEPH_STANDARD = "STDANDARD"
)

const (
	GCS_MULTI_REGIONAL = "MULTI_REGIONAL"
	GCS_REGIONAL       = "REGIONAL"
	GCS_NEARLINE       = "NEARLINE"
	GCS_COLDLINE       = "COLDLINE"
)

//Object Storage Type
const (
	OSTYPE_OPENSDS       = "OpenSDS"
	OSTYPE_AWS           = "aws-s3"
	OSTYPE_Azure         = "azure-blob"
	OSTYPE_OBS           = "hw-obs"
	OSTYPE_GCS           = "gcp-s3"
	OSTYPE_CEPH          = "ceph-s3"
	OSTYPE_FUSIONSTORAGE = "fusionstorage-object"
)

const (
	DBKEY_DELETEMARKER   = "isdeletemarker"
	DBKEY_INITFLAG       = "initflag"
	DBKEY_OBJECTKEY      = "objectkey"
	DBKEY_UPLOADID       = "uploadid"
	DBKEY_LASTMODIFIED   = "lastmodified"
	DBKEY_SUPPOSEDSTATUS = "supposedstatus"
	DBKEY_LOCKOBJ_OBJKEY = "objkey"
	DBKEY_BUCKET         = "bucket"
	DBKEY_INITTIME       = "inittime"
	DBKEY_NAME           = "name"
	DBKEY_LIFECYCLE      = "lifecycleconfiguration"
	DBKEY_ID             = "id"
)

type ObjsCountInfo struct {
	Size  int64
	Count int64
}

const (
	MaxObjectList  = 1000 // Limit number of objects in a listObjectsResponse.
	MaxUploadsList = 1000 // Limit number of uploads in a listUploadsResponse.
	MaxPartsList   = 1000 // Limit number of parts in a listPartsResponse.
)

const (
	VersioningEabled    = "Enabled"
	VersioningDisabled  = "Disabled"
	VersioningSuspended = "Suspended"
)

type ListObjsAppendInfo struct {
	Prefixes   []string
	Truncated  bool
	NextMarker string
}

const (
	MoveType_Invalid = iota
	MoveType_MoveCrossBuckets
	MoveType_ChangeLocation
	MoveType_ChangeStorageTier
)

const (
	CopySourceType_EndUser = iota
	CopySourceType_Lifecycle
	CopySourceType_Migration
)

func Md5Content(data []byte) string {
	md5Ctx := md5.New()
	md5Ctx.Write(data)
	cipherStr := md5Ctx.Sum(nil)
	//value := base64.StdEncoding.EncodeToString(cipherStr)
	value := hex.EncodeToString(cipherStr)
	return value
}
