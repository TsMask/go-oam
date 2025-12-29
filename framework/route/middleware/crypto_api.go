package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/utils/crypto"
	"github.com/tsmask/go-oam/framework/utils/parse"

	"github.com/gin-gonic/gin"
)

// CryptoApiOpt 接口加解密配置
type CryptoApiOpt struct {
	// 接口加解密是否开启
	Enable bool
	// 对请求解密
	RequestDecrypt bool
	// 对响应加密
	ResponseEncrypt bool
	// 密钥32位字符串
	KeyAES string
}

// CryptoApi 接口加解密
//
// 示例参数：middleware.CryptoApi(true, true)
//
// 参数表示：对请求解密，对响应加密
//
// 请将中间件放在最前置，对请求优先处理
func CryptoApi(opt CryptoApiOpt) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !opt.Enable {
			c.Next()
			return
		}

		// 请求解密时对请求data注入
		if opt.RequestDecrypt {
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
			dataBodyStr, err := crypto.AESDecryptBase64(contentDe, opt.KeyAES)
			if err != nil {
				c.JSON(422, resp.CodeMsg(422001, "decrypted data could not be parsed"))
				c.Abort() // 停止执行后续的处理函数
				return
			}

			// 分配回请求体
			if method == "GET" {
				var urlParams map[string]any
				json.Unmarshal([]byte(dataBodyStr), &urlParams)
				rawQuery := url.Values{}
				for k, v := range urlParams {
					rawQuery.Add(k, fmt.Sprintf("%v", v))
				}
				c.Request.URL.RawQuery = rawQuery.Encode()
			} else if contentType == gin.MIMEJSON {
				c.Request.Body = io.NopCloser(bytes.NewBuffer([]byte(dataBodyStr)))
			}
		}

		// 响应加密时替换原有的响应体
		var rbw *replaceBodyWriter
		if opt.ResponseEncrypt {
			rbw = &replaceBodyWriter{
				body:           &bytes.Buffer{},
				ResponseWriter: c.Writer,
			}
			c.Writer = rbw
		}

		// 调用下一个处理程序
		c.Next()

		// 响应加密时对响应data数据进行加密
		if opt.ResponseEncrypt {
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
						contentEn, err := crypto.AESEncryptBase64("=:)"+string(byteBodyData), opt.KeyAES)
						if err != nil {
							rbw.ReplaceWrite([]byte(fmt.Sprintf(`{"code":"%d","msg":"encrypt err"}`, resp.CODE_ERROR)))
						} else {
							// 响应加密
							byteBody, err := json.Marshal(map[string]any{
								"code": resp.CODE_ENCRYPT,
								"msg":  resp.MSG_ENCRYPT,
								"data": contentEn,
							})
							if err != nil {
								rbw.ReplaceWrite([]byte(fmt.Sprintf(`{"code":"%d","msg":"encrypt err"}`, resp.CODE_ERROR)))
							}
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
