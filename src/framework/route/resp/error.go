package resp

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FormatBindError 格式化Gin ShouldBindWith绑定错误
//
// binding:"required" 验证失败返回: field=id type=string tag=required value=
func FormatBindError(err error) string {
	if errs, ok := err.(validator.ValidationErrors); ok {
		var errMsgs []string
		for _, e := range errs {
			str := fmt.Sprintf("[field=%s, type=%s, tag=%s, param=%s, value=%v]", e.Field(), e.Type().Name(), e.Tag(), e.Param(), e.Value())
			errMsgs = append(errMsgs, str)
		}
		return strings.Join(errMsgs, ", ")
	}
	return err.Error()
}
