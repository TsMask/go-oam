package resp

// |HTTP|状态码|描述|排查建议|
// |----|----|----|----|
// |500 |500001 |internal error|服务内部错误|
// |200 |200999 |encrypt|正常请求加密数据|
// |200 |200001 |request success|正常请求成功|
// |200 |400001 |exist error|正常请求错误信息|
// |200 |400002 |ratelimit over|请求限流|
// |401 |401001 |authentication error|身份认证失败或者过期|
// |401 |401002 |authentication invalid error|无效身份信息|
// |401 |401003 |authorization token error|令牌字符为空|
// |401 |401004 |device fingerprint mismatch|设备指纹信息不匹配|
// |403 |403001 |permission error|权限未分配|
// |422 |422001 |params error|参数接收解析错误|
// |422 |422002 |params error|参数属性传入错误|

// ====== 500 ======
const (
	// CODE_ERROR_INTERNAL 响应-code服务内部错误
	CODE_INTERNAL = 500001
	// MSG_ERROR_INTERNAL 响应-msg服务内部错误
	MSG_INTERNAL = "internal error"
)

// ====== 200 ======
const (
	// CODE_ENCRYPT 响应-code加密数据
	CODE_ENCRYPT = 200999
	// MSG_ENCRYPT 响应-msg加密数据
	MSG_ENCRYPT = "encrypt"

	// CODE_SUCCESS 响应-code正常成功
	CODE_SUCCESS = 200001
	// MSG_SUCCCESS 响应-msg正常成功
	MSG_SUCCCESS = "success"

	// CODE_ERROR 响应-code错误失败
	CODE_ERROR = 400001
	// MSG_ERROR 响应-msg错误失败
	MSG_ERROR = "error"
)

// ====== 401 ======
const (
	// CODE_ERROR 响应-code身份认证失败或者过期
	CODE_AUTH = 401001

	// CODE_AUTH_INVALID 响应-code无效身份信息
	CODE_AUTH_INVALID = 401002
)

// ====== 422 ======
const (
	// CODE_PARAM_PARSER 响应-code参数接收解析错误
	CODE_PARAM_PARSER = 422001
	// CODE_PARAM_CHEACK 响应-code参数属性传入错误
	CODE_PARAM_CHEACK = 422002
)
