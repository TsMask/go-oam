package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/framework/logger"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/utils/crypto"
	"github.com/tsmask/go-oam/framework/utils/parse"

	"github.com/gin-gonic/gin"
)

// CryptoApi 接口加解密
//
// 示例参数：middleware.CryptoApi(true, true)
//
// 参数表示：对请求解密，对响应加密
//
// 请将中间件放在最前置，对请求优先处理
func CryptoApi(requestDecrypt, responseEncrypt bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 登录认证，默认打开
		enable := parse.Boolean(config.Get("cryptoapi"))
		if !enable {
			c.Next()
			return
		}

		// 请求解密时对请求data注入
		if requestDecrypt {
			method := c.Request.Method
			contentType := c.ContentType()
			contentDe := ""
			// 请求参数解析
			if method == "GET" {
				contentDe = c.Query("data")
			} else if contentType == gin.MIMEJSON {
				var body struct {
					Data string `json:"data" binding:"required"`
				}
				if err := c.ShouldBindJSON(&body); err == nil {
					contentDe = body.Data
				}
			}

			// 是否存在data字段数据
			if contentDe == "" {
				c.JSON(422, resp.CodeMsg(422002, "decrypt not found field data"))
				c.Abort() // 停止执行后续的处理函数
				return
			}

			// 解密-原数据加密前含16位长度iv
			apiKey := config.Get("aes.apiKey").(string)
			dataBodyStr, err := crypto.AESDecryptBase64(contentDe, apiKey)
			if err != nil {
				logger.Errorf("CryptoApi decrypt err => %v", err)
				c.JSON(422, resp.CodeMsg(422001, "decrypted data could not be parsed"))
				c.Abort() // 停止执行后续的处理函数
				return
			}

			// 分配回请求体
			if method == "GET" {
				var urlParams map[string]any
				json.Unmarshal([]byte(dataBodyStr), &urlParams)
				rawQuery := []string{}
				for k, v := range urlParams {
					rawQuery = append(rawQuery, fmt.Sprintf("%s=%v", k, v))
				}
				c.Request.URL.RawQuery = strings.Join(rawQuery, "&")
			} else if contentType == gin.MIMEJSON {
				c.Request.Body = io.NopCloser(bytes.NewBuffer([]byte(dataBodyStr)))
			}
		}

		// 响应加密时替换原有的响应体
		var rbw *replaceBodyWriter
		if responseEncrypt {
			rbw = &replaceBodyWriter{
				body:           &bytes.Buffer{},
				ResponseWriter: c.Writer,
			}
			c.Writer = rbw
		}

		// 调用下一个处理程序
		c.Next()

		// 响应加密时对响应data数据进行加密
		if responseEncrypt {
			// 满足成功并带数据的响应进行加密
			if c.Writer.Status() == 200 {
				var resBody map[string]any
				json.Unmarshal(rbw.body.Bytes(), &resBody)
				codeV, codeOk := resBody["code"]
				dataV, dataOk := resBody["data"]
				if codeOk && dataOk {
					if parse.Number(codeV) == resp.CODE_SUCCESS {
						byteBodyData, _ := json.Marshal(dataV)
						// 加密-原数据头加入标记16位长度iv终止符
						apiKey := config.Get("aes.apiKey").(string)
						contentEn, err := crypto.AESEncryptBase64("=:)"+string(byteBodyData), apiKey)
						if err != nil {
							logger.Errorf("CryptoApi encrypt err => %v", err)
							rbw.ReplaceWrite([]byte(fmt.Sprintf(`{"code":"%d","msg":"encrypt err"}`, resp.CODE_ERROR)))
						} else {
							// 响应加密
							byteBody, _ := json.Marshal(map[string]any{
								"code": resp.CODE_ENCRYPT,
								"msg":  resp.MSG_ENCRYPT,
								"data": contentEn,
							})
							rbw.ReplaceWrite(byteBody)
						}
					}
				} else {
					rbw.ReplaceWrite(nil)
				}
			} else {
				rbw.ReplaceWrite(nil)
			}
		}
		//
	}
}

// replaceBodyWriter 替换默认的响应体
type replaceBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 写入响应体
func (r replaceBodyWriter) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

// ReplaceWrite 替换响应体
func (r *replaceBodyWriter) ReplaceWrite(b []byte) (int, error) {
	if b == nil {
		return r.ResponseWriter.Write(r.body.Bytes())
	}
	r.body = &bytes.Buffer{}
	r.body.Write(b)
	return r.ResponseWriter.Write(r.body.Bytes())
}
