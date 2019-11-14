package datastore

import (
	_ "github.com/opensds/multi-cloud/s3/pkg/datastore/aws"
	_ "github.com/opensds/multi-cloud/s3/pkg/datastore/azure"
	_ "github.com/opensds/multi-cloud/s3/pkg/datastore/ceph"
	_ "github.com/opensds/multi-cloud/s3/pkg/datastore/huawei"
	_ "github.com/opensds/multi-cloud/s3/pkg/datastore/ibm"
	_ "github.com/opensds/multi-cloud/s3/pkg/datastore/yig"
)
