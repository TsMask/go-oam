package push

// ============ 历史查询 API ============

// HistoryList 获取指定类型的推送历史
//
//	n < 0: 返回空
//	n == 0: 返回全部
//	n > 0: 返回最近 n 条
func (p *Push) HistoryList(typeStr string, n int) []Record {
	records := p.hist.HistoryList(typeStr, n)
	result := make([]Record, len(records))
	for i, r := range records {
		result[i] = Record{
			NeUID:      r.NeUID,
			RecordTime: r.RecordTime,
			RecordType: r.RecordType,
			RecordData: r.RecordData,
		}
	}
	return result
}

// HistorySetSizeByType 设置指定类型历史记录上限
func (p *Push) HistorySetSizeByType(typeStr string, size int) {
	p.hist.HistorySetSizeByType(typeStr, size)
}

// HistoryClear 清空指定类型的历史
func (p *Push) HistoryClear(typeStr string) {
	p.hist.HistoryClear(typeStr)
}

// HistorySetSize 设置所有历史记录上限
func (p *Push) HistorySetSize(size int) {
	p.hist.HistorySetSize(size)
}

// HistoryTypes 获取所有已有历史记录的类型
func (p *Push) HistoryTypes() []string {
	return p.hist.HistoryTypes()
}
