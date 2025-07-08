package reqctx

import (
	"strings"

	"github.com/tsmask/go-oam/src/framework/constants"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

// QueryMap Query参数转换Map
func QueryMap(c *gin.Context) map[string]string {
	queryValues := c.Request.URL.Query()
	queryParams := make(map[string]string, len(queryValues))
	for key, values := range queryValues {
		queryParams[key] = values[0]
	}
	return queryParams
}

// BodyJSONMap JSON参数转换Map
func BodyJSONMap(c *gin.Context) map[string]any {
	params := make(map[string]any, 0)
	c.ShouldBindBodyWithJSON(&params)
	return params
}

// RequestParamsMap 请求参数转换Map
func RequestParamsMap(c *gin.Context) map[string]any {
	params := make(map[string]any, 0)
	// json
	if strings.HasPrefix(c.ContentType(), "application/json") {
		c.ShouldBindBodyWithJSON(&params)
	}

	// 表单
	formParams := c.Request.PostForm
	for key, value := range formParams {
		if _, ok := params[key]; !ok {
			params[key] = value[0]
		}
	}

	// 查询
	queryParams := c.Request.URL.Query()
	for key, value := range queryParams {
		if _, ok := params[key]; !ok {
			params[key] = value[0]
		}
	}
	return params
}

// Authorization 解析请求头
func Authorization(c *gin.Context) string {
	// Query请求查询
	if authQuery, ok := c.GetQuery(constants.ACCESS_TOKEN); ok && authQuery != "" {
		return authQuery
	}
	// Header请求头
	if authHeader := c.GetHeader(constants.ACCESS_TOKEN); authHeader != "" {
		return authHeader
	}

	// Query请求查询
	if authQuery, ok := c.GetQuery(constants.ACCESS_TOKEN_QUERY); ok && authQuery != "" {
		return authQuery
	}
	// Header请求头
	authHeader := c.GetHeader(constants.HEADER_KEY)
	if authHeader == "" {
		return ""
	}
	// 拆分 Authorization 请求头，提取 JWT 令牌部分
	tokenStr := strings.Replace(authHeader, constants.HEADER_PREFIX, "", 1)
	if len(tokenStr) > 64 {
		return strings.TrimSpace(tokenStr) // 去除可能存在的空格
	}
	return ""
}

// AcceptLanguage 解析客户端接收语言 zh：中文 en: 英文
func AcceptLanguage(c *gin.Context) string {
	preferredLanguage := language.English

	// Query请求查询
	if v, ok := c.GetQuery("language"); ok && v != "" {
		tags, _, _ := language.ParseAcceptLanguage(v)
		if len(tags) > 0 {
			preferredLanguage = tags[0]
		}
	}
	//  Header请求头
	if v := c.GetHeader("Accept-Language"); v != "" {
		tags, _, _ := language.ParseAcceptLanguage(v)
		if len(tags) > 0 {
			preferredLanguage = tags[0]
		}
	}

	// 只取前缀
	lang := preferredLanguage.String()
	arr := strings.Split(lang, "-")
	return arr[0]
}
