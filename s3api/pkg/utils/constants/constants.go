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

package constants

const (
	MaxObjectSize           = 5 * 1024 * 1024 * 1024 // 5GB
)

const (
	KLastModified = "lastmodified"
	KObjKey       = "objkey"
	KStorageTier  = "tier"
	KPrefix       = "prefix"
	KMarker       = "marker"
	KDelimiter    = "delimiter"
	KVerMarker    = "verMarker"
)

const (
	REQUEST_PATH_BUCKET_NAME         = "bucketName"
	REQUEST_PATH_OBJECT_KEY          = "objectKey"
	REQUEST_HEADER_CONTENT_LENGTH    = "Content-Length"
	REQUEST_HEADER_STORAGE_CLASS     = "x-amz-storage-class"
	REQUEST_HEADER_COPY_SOURCE       = "X-Amz-Copy-Source"
	REQUEST_HEADER_COPY_SOURCE_RANGE = "X-Amz-Copy-Source-Range"
	REQUEST_HEADER_ACL               = "X-Amz-Acl"
	REQUEST_HEADER_CONTENT_MD5       = "Content-Md5"
	REQUEST_HEADER_CONTENT_TYPE      = "Content-Type"
)

const (
	REQUEST_FORM_KEY    = "Key"
	REQUEST_FORM_BUCKET = "Bucket"
)

const (
	StorageClassOpenSDSStandard = "STANDARD"
	StorageClassAWSStandard     = "STANDARD"
)

const (
	ActionNameExpiration = "expiration"
	ActionNameTransition = "transition"
)

const (
	ExpirationMinDays           = 1
	TransitionMinDays           = 30
	LifecycleTransitionDaysStep = 30 // The days an object should be save in the current tier before transition to the next tier
	TransitionToArchiveMinDays  = 1
)

const (
	ListObjectsType2Str string = "2"
	ListObjectsType2Int int32  = 2
	ListObjectsType1Int int32  = 1
)
