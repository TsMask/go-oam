package reqctx

import (
	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/modules/callback"

	"github.com/gin-gonic/gin"
)

const ConfigKey = "oam:ctx:config"

// ConfigInContext 将配置实例设置到 Gin 上下文中
func ConfigInContext(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ConfigKey, cfg)
		c.Next()
	}
}

// OAMConfig 从 Gin 上下文中获取配置实例
func OAMConfig(c *gin.Context) *config.Config {
	if val, exists := c.Get(ConfigKey); exists {
		if cfg, ok := val.(*config.Config); ok {
			return cfg
		}
	}
	return config.New()
}

const CallbackKey = "oam:ctx:callback"

// CallbackInContext 将回调实例设置到 Gin 上下文中
func CallbackInContext(handler callback.CallbackHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(CallbackKey, handler)
		c.Next()
	}
}

// OAMCallback 从 Gin 上下文中获取回调实例
func OAMCallback(c *gin.Context) callback.CallbackHandler {
	if val, exists := c.Get(CallbackKey); exists {
		if handler, ok := val.(callback.CallbackHandler); ok {
			return handler
		}
	}
	return nil
}
