package model

// Config 网元配置信息对象
type Config struct {
	ParamName  string `json:"paramName" form:"paramName" binding:"required"` // 参数名称
	ParamValue any    `json:"paramValue" form:"paramValue"`                  // 参数值
	Loc        string `json:"loc" form:"loc"`                                // 定位符 list为空字符 array使用与数据对象内index一致,有多层时划分嵌套层(index/subParamName/index)
}
