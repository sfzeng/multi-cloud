package constants

const (
	//Signature parameter name
	AuthorizationHeader = "Authorization"
	SignDateHeader      = "X-Auth-Date"

	// Token parameter name
	AuthTokenHeader   = "X-Auth-Token"
)

const (
	CTX_KEY_TENANT_ID   = "Tenantid"
	CTX_KEY_USER_ID     = "Userid"
	CTX_KEY_IS_ADMIN    = "Isadmin"
	CTX_VAL_TRUE        = "true"
	CTX_REPRE_TENANT    = "Representedtenantid"
	CTX_KEY_OBJECT_KEY  = "ObjectKey"
	CTX_KEY_BUCKET_NAME = "BucketName"
	CTX_KEY_SIZE        = "ObjectSize"
	CTX_KEY_LOCATION    = "Location"
)

const (
	ListObjectsType2Str string = "2"
	ListObjectsType2Int int32  = 2
	ListObjectsType1Int int32  = 1
)

