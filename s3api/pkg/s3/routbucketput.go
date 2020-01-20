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

package s3

import (
	"github.com/emicklei/go-restful"
	log "github.com/sirupsen/logrus"
)

func (s *APIService) RouteBucketPut(request *restful.Request, response *restful.Response) {
	log.Debugf("put bucket, request URL:%v\n", *request.Request.URL)
	/*if !policy.Authorize(request, response, "bucket:put") {
		log.Errorln("Authorize policy check failed.")
		WriteErrorResponse(response, request, s3error.ErrAccessDenied)
		return
	}*/

	if IsQuery(request, "acl") {
		s.BucketAclPut(request, response)
	} else if IsQuery(request, "versioning") {
		s.BucketVersioningPut(request, response)
	} else if IsQuery(request, "website") {
		//TODO
	} else if IsQuery(request, "cors") {
		//TODO

	} else if IsQuery(request, "replication") {
		//TODO

	} else if IsQuery(request, "lifecycle") {
		s.BucketLifecyclePut(request, response)
	}  else if IsQuery(request, "DefaultEncryption") {
		s.BucketSSEPut(request, response)
	} else {
		s.BucketPut(request, response)
	}
}
