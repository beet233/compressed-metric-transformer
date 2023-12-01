package main

import (
	"github.com/prometheus/common/expfmt"
)

// TODO process 方法，其中 readIn 过程调用到 reader，如果 magic 是 meta 则更新 meta
type CompressProcessor struct {
	metadata expfmt.Metadata
}

var processor *CompressProcessor = &CompressProcessor{expfmt.Metadata{MetricFamilyMap: make(map[uint64]expfmt.MetricFamilyMetadata), ReverseMetricFamilyMap: make(map[string]uint64)}}

func GetCompressProcessor() *CompressProcessor {
	return processor
}

func (p *CompressProcessor) process(data []byte) ([]byte, error) {
	return data, nil
}
