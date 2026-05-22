package state

import (
	"context"

	"github.com/shirou/gopsutil/v4/disk"
)

// Disk 磁盘分区详细信息
type Disk struct {
	Device  string  `json:"device"`  // 设备路径
	Total   uint64  `json:"total"`   // 总容量，单位字节
	Used    uint64  `json:"used"`    // 已用
	Avail   uint64  `json:"avail"`   // 可用
	Percent float64 `json:"percent"` // 使用率，单位 %
}

// SystemDisk 获取磁盘分区使用详情。
// ctx 用于控制超时，建议设置 300ms 以上。
func SystemDisk(ctx context.Context) []Disk {
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil
	}
	result := make([]Disk, 0, len(partitions))
	for _, p := range partitions {
		usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil {
			continue
		}
		result = append(result, Disk{
			Device:  p.Device,
			Total:   usage.Total,
			Used:    usage.Used,
			Avail:   usage.Free,
			Percent: usage.UsedPercent,
		})
	}
	return result
}
