package registry

import (
	"sync"

	"github.com/tsmask/go-oam/nms/internal/session"
)

// ============================================
// Registry NE 连接注册中心
// ============================================

type Registry struct {
	mu  sync.RWMutex
	nes map[string]*NE // key: ne_id

	// 回调函数
	onRegister   func(*NE)
	onUnregister func(*NE)
	onUpdate     func(*NE)
}

type NE struct {
	ID          string
	Type        string
	IP          string
	Port        int32
	Attrs       map[string]string
	Capabilities map[string]string
	ConnectedAt int64   // 毫秒
	SessionID   string
	Online      bool
}

// ============================================
// 构造函数
// ============================================

func New() *Registry {
	return &Registry{
		nes: make(map[string]*NE),
	}
}

// ============================================
// 注册/注销
// ============================================

// Register 注册 NE
func (r *Registry) Register(ctx *session.Context) *NE {
	r.mu.Lock()
	defer r.mu.Unlock()

	ne := &NE{
		ID:          ctx.NEID,
		Type:        ctx.NEType,
		IP:          ctx.IP,
		Port:        ctx.Port,
		Attrs:       ctx.Attrs,
		ConnectedAt: ctx.ConnectedAt,
		SessionID:   ctx.SessionID,
		Online:      true,
	}

	r.nes[ctx.NEID] = ne

	if r.onRegister != nil {
		r.onRegister(ne)
	}

	return ne
}

// Unregister 注销 NE
func (r *Registry) Unregister(neID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ne, ok := r.nes[neID]; ok {
		ne.Online = false
		if r.onUnregister != nil {
			r.onUnregister(ne)
		}
	}
	delete(r.nes, neID)
}

// Update 更新 NE 信息
func (r *Registry) Update(neID string, updateFn func(*NE)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ne, ok := r.nes[neID]; ok {
		updateFn(ne)
		if r.onUpdate != nil {
			r.onUpdate(ne)
		}
	}
}

// ============================================
// 查询
// ============================================

// Get 获取 NE
func (r *Registry) Get(neID string) *NE {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.nes[neID]
}

// List 获取所有 NE
func (r *Registry) List() []*NE {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*NE, 0, len(r.nes))
	for _, ne := range r.nes {
		result = append(result, ne)
	}
	return result
}

// ListOnline 获取所有在线 NE
func (r *Registry) ListOnline() []*NE {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*NE, 0)
	for _, ne := range r.nes {
		if ne.Online {
			result = append(result, ne)
		}
	}
	return result
}

// ListByType 按类型获取 NE
func (r *Registry) ListByType(neType string) []*NE {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*NE, 0)
	for _, ne := range r.nes {
		if ne.Type == neType && ne.Online {
			result = append(result, ne)
		}
	}
	return result
}

// Count 获取 NE 总数
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.nes)
}

// CountOnline 获取在线 NE 总数
func (r *Registry) CountOnline() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, ne := range r.nes {
		if ne.Online {
			count++
		}
	}
	return count
}

// ============================================
// 回调设置
// ============================================

// SetOnRegister 设置注册回调
func (r *Registry) SetOnRegister(fn func(*NE)) {
	r.onRegister = fn
}

// SetOnUnregister 设置注销回调
func (r *Registry) SetOnUnregister(fn func(*NE)) {
	r.onUnregister = fn
}

// SetOnUpdate 设置更新回调
func (r *Registry) SetOnUpdate(fn func(*NE)) {
	r.onUpdate = fn
}

// ============================================
// 工具方法
// ============================================

// Clear 清空注册表
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nes = make(map[string]*NE)
}

// Close 关闭注册中心（清空并触发回调）
func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for neID := range r.nes {
		if ne := r.nes[neID]; r.onUnregister != nil {
			r.onUnregister(ne)
		}
		delete(r.nes, neID)
	}
}