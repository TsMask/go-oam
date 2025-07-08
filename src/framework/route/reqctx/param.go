package reqctx

import (
	"fmt"

	"github.com/tsmask/go-oam/src/framework/utils/crypto"

	"github.com/gin-gonic/gin"
)

// DeviceFingerprint 设备指纹信息
func DeviceFingerprint(c *gin.Context, v any) string {
	str := fmt.Sprintf("%v:%s", v, c.Request.UserAgent())
	return crypto.SHA256ToBase64(str)
}
