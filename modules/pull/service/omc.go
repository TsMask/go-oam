package service

import (
	"fmt"

	"github.com/tsmask/go-oam/framework/config"
	"github.com/tsmask/go-oam/modules/pull/model"
)

// OMCInfoSet 网管信息设置
func OMCInfoSet(v model.OMC) error {
	config.Set("omc", map[string]any{
		"url":     v.Url,
		"neuid":   v.NeUID,
		"coreuid": v.CoreUID,
	})
	return nil
}

// OMCInfoGet 网管信息获取
func OMCInfoGet() model.OMC {
	v, ok := config.Get("omc").(map[string]any)
	if !ok {
		return model.OMC{}
	}
	return model.OMC{
		Url:     fmt.Sprint(v["url"]),
		NeUID:   fmt.Sprint(v["neuid"]),
		CoreUID: fmt.Sprint(v["coreuid"]),
	}
}
