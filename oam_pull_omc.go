package oam

import (
	pullModel "github.com/tsmask/go-oam/modules/pull/model"
	pullService "github.com/tsmask/go-oam/modules/pull/service"
)

// OMC 网管信息
type OMC = pullModel.OMC

// OMCInfoGet 网管信息获取
func OMCInfoGet() OMC {
	return pullService.OMCInfoGet()
}

// OMCInfoSet 网管信息设置
func OMCInfoSet(info OMC) {
	pullService.OMCInfoSet(info)
}
