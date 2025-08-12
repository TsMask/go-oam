package security

import "github.com/gin-gonic/gin"

// CSRF 配置，包含是否启用、类型和 Referer 白名单。
type CSRF struct {
	Enable           bool     // 是否启用 CSRF 防护
	Type             string   // CSRF 类型
	RefererWhiteList []string // Referer 白名单
}

// XFrame 配置，控制 X-Frame-Options 响应头。
type XFrame struct {
	Enable bool   // 是否启用 X-Frame-Options
	Value  string // X-Frame-Options 的值
}

// CSP 配置，控制 Content-Security-Policy 响应头。
type CSP struct {
	Enable bool // 是否启用 CSP
}

// HSTS 配置，控制 Strict-Transport-Security 响应头。
type HSTS struct {
	Enable            bool // 是否启用 HSTS
	MaxAge            int  // HSTS 的最大存活时间（秒）
	IncludeSubdomains bool // 是否包含子域名
}

// NoOpen 配置，控制 X-Download-Options 响应头。
type NoOpen struct {
	Enable bool // 是否启用 X-Download-Options
}

// NoSniff 配置，控制 X-Content-Type-Options 响应头。
type NoSniff struct {
	Enable bool // 是否启用 X-Content-Type-Options
}

// XSSProtection 配置，控制 X-XSS-Protection 响应头。
type XSSProtection struct {
	Enable bool   // 是否启用 X-XSS-Protection
	Value  string // X-XSS-Protection 的值
}

// SecurityOpt 定义了安全相关的配置选项。
// 包含 CSRF、X-Frame-Options、CSP、HSTS、X-Download-Options、X-Content-Type-Options、X-XSS-Protection 等安全头的开关和参数。
type SecurityOpt struct {
	CSRF          CSRF          // CSRF 配置
	XFrame        XFrame        // X-Frame-Options 配置
	CSP           CSP           // CSP 配置
	HSTS          HSTS          // HSTS 配置
	NoOpen        NoOpen        // X-Download-Options 配置
	NoSniff       NoSniff       // X-Content-Type-Options 配置
	XSSProtection XSSProtection // X-XSS-Protection 配置
}

// SecurityDefaultOpt 安全默认配置
var SecurityDefaultOpt = SecurityOpt{
	CSRF: CSRF{
		Enable:           false,
		Type:             "referer",
		RefererWhiteList: []string{"127.0.0.1:33030"},
	},
	XFrame: XFrame{
		Enable: false,
		Value:  "SAMEORIGIN",
	},
	CSP: CSP{
		Enable: true,
	},
	HSTS: HSTS{
		Enable:            false,
		MaxAge:            31536000,
		IncludeSubdomains: false,
	},
	NoOpen: NoOpen{
		Enable: false,
	},
	NoSniff: NoSniff{
		Enable: false,
	},
	XSSProtection: XSSProtection{
		Enable: true,
		Value:  "1; mode=block",
	},
}

// Security 安全
func Security(opt SecurityOpt) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 拦截，判断是否有效Referer
		referer(c, opt.CSRF)

		// 无拦截，仅仅设置响应头
		xframe(c, opt.XFrame)
		csp(c, opt.CSP)
		hsts(c, opt.HSTS)
		noopen(c, opt.NoOpen)
		nosniff(c, opt.NoSniff)
		xssProtection(c, opt.XSSProtection)

		c.Next()
	}
}
