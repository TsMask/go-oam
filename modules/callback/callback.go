package callback

// 外部回调接口
type CallbackHandler interface {
	// 备用状态
	Standby() bool
	// Redis 实例 *redis.Client
	Redis() any
	// Telnet 消息处理
	Telnet(command string) string
	// SNMP 消息处理
	SNMP(oid, operType string, value any) any
	// Config 网元配置消息处理
	// action  操作行为 C/U/R/D Create（创建）、Update（更新）、Read（读取）、Delete（删除）
	// paramName 参数名称
	// loc 定位符 list为空字符 array使用与数据对象内index一致,有多层时划分嵌套层(index/subParamName/index)
	// paramValue 参数值
	Config(action, paramName, loc string, paramValue any) error
}

// CallbackFuncs 提供基于函数的回调实现，方便按需注入
type CallbackFuncs struct {
	OnStandby func() bool
	OnRedis   func() any
	OnTelnet  func(command string) string
	OnSNMP    func(oid, operType string, value any) any
	OnConfig  func(action, paramName, loc string, paramValue any) error
}

// Standby 备用状态
func (f *CallbackFuncs) Standby() bool {
	if f.OnStandby != nil {
		return f.OnStandby()
	}
	return false
}

// Redis 实例 *redis.Client
func (f *CallbackFuncs) Redis() any {
	if f.OnRedis != nil {
		return f.OnRedis()
	}
	return nil
}

// Telnet 消息处理
func (f *CallbackFuncs) Telnet(command string) string {
	if f.OnTelnet != nil {
		return f.OnTelnet(command)
	}
	return "telnet unrealized"
}

// SNMP 消息处理
func (f *CallbackFuncs) SNMP(oid, operType string, value any) any {
	if f.OnSNMP != nil {
		return f.OnSNMP(oid, operType, value)
	}
	return "snmp unrealized"
}

// Config 网元配置消息处理
func (f *CallbackFuncs) Config(action, paramName, loc string, paramValue any) error {
	if f.OnConfig != nil {
		return f.OnConfig(action, paramName, loc, paramValue)
	}
	return nil
}
