package resp

// Resp 响应结构体
type Resp struct {
	Code int    `json:"code"`           // 响应状态码
	Msg  string `json:"msg"`            // 响应信息
	Data any    `json:"data,omitempty"` // 响应数据
}

// CodeMsg 响应结果
func CodeMsg(code int, msg string) Resp {
	return Resp{Code: code, Msg: msg}
}

// Ok 响应成功结果
func Ok(v map[string]any) map[string]any {
	args := make(map[string]any)
	args["code"] = CODE_SUCCESS
	args["msg"] = MSG_SUCCCESS
	// v合并到args
	for key, value := range v {
		args[key] = value
	}
	return args
}

// OkMsg 响应成功结果信息
func OkMsg(msg string) Resp {
	return Resp{Code: CODE_SUCCESS, Msg: msg}
}

// OkData 响应成功结果数据
func OkData(data any) Resp {
	return Resp{Code: CODE_SUCCESS, Msg: MSG_SUCCCESS, Data: data}
}

// Err 响应失败结果 map[string]any{}
func Err(v map[string]any) map[string]any {
	args := make(map[string]any)
	args["code"] = CODE_ERROR
	args["msg"] = MSG_ERROR
	// v合并到args
	for key, value := range v {
		args[key] = value
	}
	return args
}

// ErrMsg 响应失败结果信息
func ErrMsg(msg string) Resp {
	return Resp{Code: CODE_ERROR, Msg: msg}
}

// ErrData 响应失败结果数据
func ErrData(data any) Resp {
	return Resp{Code: CODE_ERROR, Msg: MSG_ERROR, Data: data}
}
