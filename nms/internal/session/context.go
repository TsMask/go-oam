package session

// ============================================
// 辅助函数
// ============================================

// IsActive 检查会话是否活跃
func (c *Context) IsActive() bool {
	return c.Status == SessionStatusActive || c.Status == SessionStatusConnected
}

// IsTimeout 检查会话是否超时（心跳超时 60 秒）
func (c *Context) IsTimeout() bool {
	return nowMs()-c.LastHeartbeat > 60*1000
}

// CopyContext 复制会话上下文（深拷贝）
func CopyContext(c *Context) *Context {
	attrs := make(map[string]string)
	for k, v := range c.Attrs {
		attrs[k] = v
	}

	return &Context{
		ID:            c.ID,
		NEID:          c.NEID,
		SessionID:     c.SessionID,
		NEType:        c.NEType,
		IP:            c.IP,
		Port:          c.Port,
		Attrs:         attrs,
		ConnectedAt:   c.ConnectedAt,
		LastHeartbeat: c.LastHeartbeat,
		Status:        c.Status,
	}
}