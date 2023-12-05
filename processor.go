package main

import (
	"bytes"
	"errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"math"
	"strconv"
	"strings"
	"sync"
)

type CompressProcessor struct {
	metadata expfmt.Metadata
}

var processor = &CompressProcessor{expfmt.Metadata{MetricFamilyMap: make(map[uint64]expfmt.MetricFamilyMetadata), ReverseMetricFamilyMap: make(map[string]uint64)}}

var (
	numBufPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 0, 24)
			return &b
		},
	}
)

func GetCompressProcessor() *CompressProcessor {
	return processor
}

func (p *CompressProcessor) process(data []byte) ([]byte, error) {
	// result := make([]byte, 0, 1024)
	result := bytes.NewBuffer(make([]byte, 0, 1024))
	reader := NewDataReader(data)
	if len(data) >= 7 && string(data[:7]) == "cprmeta" {
		// skip magic number
		reader.readString(7)
		// read version
		version, err := reader.readLeb128Int()
		if err != nil {
			return nil, err
		}
		newMetadata := expfmt.Metadata{MetricFamilyMap: make(map[uint64]expfmt.MetricFamilyMetadata), ReverseMetricFamilyMap: make(map[string]uint64)}
		newMetadata.Version = version
		// read metric family
		metricFamilyLen, err := reader.readLeb128Int()
		if err != nil {
			return nil, err
		}
		for i := 0; uint64(i) < metricFamilyLen; i++ {
			metricFamilyMetadata := expfmt.MetricFamilyMetadata{LabelMap: make(map[uint64]string), ReverseLabelMap: make(map[string]uint64)}
			// read metric family type
			metricTypeEnum, err := reader.readLeb128Int()
			if err != nil {
				return nil, err
			}
			metricFamilyMetadata.MetricType = dto.MetricType(metricTypeEnum)
			// read metric family name
			nameLen, err := reader.readLeb128Int()
			if err != nil {
				return nil, err
			}
			name, err := reader.readString(int(nameLen))
			if err != nil {
				return nil, err
			}
			metricFamilyMetadata.Name = name
			// read metric family help
			helpLen, err := reader.readLeb128Int()
			if err != nil {
				return nil, err
			}
			help, err := reader.readString(int(helpLen))
			if err != nil {
				return nil, err
			}
			metricFamilyMetadata.Help = help
			// read labels
			labelLen, err := reader.readLeb128Int()
			if err != nil {
				return nil, err
			}
			for j := 0; uint64(j) < labelLen; j++ {
				labelNameLen, err := reader.readLeb128Int()
				if err != nil {
					return nil, err
				}
				labelName, err := reader.readString(int(labelNameLen))
				if err != nil {
					return nil, err
				}
				metricFamilyMetadata.LabelMap[uint64(j)] = labelName
			}
			newMetadata.MetricFamilyMap[uint64(i)] = metricFamilyMetadata
		}
		// totally replace metadata
		p.metadata = newMetadata
	}
	valMagicNum, err := reader.readString(6)
	if err != nil {
		return nil, err
	}
	if valMagicNum != "cprval" {
		return nil, errors.New("invalid magic num")
	}
	// read version
	version, err := reader.readLeb128Int()
	if err != nil {
		return nil, err
	}
	if version != p.metadata.Version {
		return nil, errors.New("invalid metadata version in cprval")
	}
	metricFamilyLen, err := reader.readLeb128Int()
	if err != nil {
		return nil, err
	}
	for i := 0; uint64(i) < metricFamilyLen; i++ {
		metricFamilyIndex, err := reader.readLeb128Int()
		if err != nil {
			return nil, err
		}
		metricFamilyMetadata := p.metadata.MetricFamilyMap[metricFamilyIndex]
		// write name type help
		result.WriteString("# HELP " + metricFamilyMetadata.Name + " " + metricFamilyMetadata.Help + "\n")
		result.WriteString("# TYPE " + metricFamilyMetadata.Name + " " + strings.ToLower(metricFamilyMetadata.MetricType.String()) + "\n")
		// read metrics
		metricLen, err := reader.readLeb128Int()
		if err != nil {
			return nil, err
		}
		for j := 0; uint64(j) < metricLen; j++ {
			labelLen, err := reader.readLeb128Int()
			if err != nil {
				return nil, err
			}
			labels := make([]uint64, labelLen)
			labelValues := make([]string, labelLen)
			for k := 0; uint64(k) < labelLen; k++ {
				labelIndex, err := reader.readLeb128Int()
				if err != nil {
					return nil, err
				}
				labels[k] = labelIndex
				labelValueLen, err := reader.readLeb128Int()
				if err != nil {
					return nil, err
				}
				labelValue, err := reader.readString(int(labelValueLen))
				labelValues[k] = labelValue
			}
			switch metricFamilyMetadata.MetricType {
			case dto.MetricType_COUNTER:
				fallthrough
			case dto.MetricType_GAUGE:
				fallthrough
			case dto.MetricType_UNTYPED:
				metricValue, err := reader.readFloat()
				if err != nil {
					return nil, err
				}
				result.WriteString(metricFamilyMetadata.Name)
				if labelLen > 0 {
					result.WriteString("{")
					for k := 0; uint64(k) < labelLen; k++ {
						result.WriteString(metricFamilyMetadata.LabelMap[labels[k]] + "=\"")
						result.WriteString(labelValues[k])
						result.WriteString("\"")
						if uint64(k) < labelLen-1 {
							result.WriteString(",")
						}
					}
					result.WriteString("}")
				}
				result.WriteString(" ")
				writeFloat(result, metricValue)
				result.WriteString("\n")
			case dto.MetricType_SUMMARY:
				quantileLen, err := reader.readLeb128Int()
				if err != nil {
					return nil, err
				}
				for k := 0; uint64(k) < quantileLen; k++ {
					quantile, err := reader.readFloat()
					if err != nil {
						return nil, err
					}
					metricValue, err := reader.readFloat()
					if err != nil {
						return nil, err
					}
					result.WriteString(metricFamilyMetadata.Name)
					result.WriteString("{")
					for m := 0; uint64(m) < labelLen; m++ {
						result.WriteString(metricFamilyMetadata.LabelMap[labels[m]] + "=\"")
						result.WriteString(labelValues[m])
						result.WriteString("\"")
						result.WriteString(",")
					}
					result.WriteString("quantile=\"")
					writeFloat(result, quantile)
					result.WriteString("\"")
					result.WriteString("} ")
					writeFloat(result, metricValue)
					result.WriteString("\n")
				}
				result.WriteString(metricFamilyMetadata.Name + "_sum")
				if labelLen > 0 {
					result.WriteString("{")
					for m := 0; uint64(m) < labelLen; m++ {
						result.WriteString(metricFamilyMetadata.LabelMap[labels[m]] + "=\"")
						result.WriteString(labelValues[m])
						result.WriteString("\"")
						if uint64(m) < labelLen-1 {
							result.WriteString(",")
						}
					}
					result.WriteString("}")
				}
				result.WriteString(" ")
				sumValue, err := reader.readFloat()
				if err != nil {
					return nil, err
				}
				writeFloat(result, sumValue)
				result.WriteString("\n")
				result.WriteString(metricFamilyMetadata.Name + "_count")
				if labelLen > 0 {
					result.WriteString("{")
					for m := 0; uint64(m) < labelLen; m++ {
						result.WriteString(metricFamilyMetadata.LabelMap[labels[m]] + "=\"")
						result.WriteString(labelValues[m])
						result.WriteString("\"")
						if uint64(m) < labelLen-1 {
							result.WriteString(",")
						}
					}
					result.WriteString("}")
				}
				result.WriteString(" ")
				countValue, err := reader.readLeb128Int()
				if err != nil {
					return nil, err
				}
				writeFloat(result, float64(countValue))
				result.WriteString("\n")
			case dto.MetricType_HISTOGRAM:
				bucketLen, err := reader.readLeb128Int()
				if err != nil {
					return nil, err
				}
				for k := 0; uint64(k) < bucketLen; k++ {
					bucket, err := reader.readFloat()
					if err != nil {
						return nil, err
					}
					metricValue, err := reader.readFloat()
					if err != nil {
						return nil, err
					}
					result.WriteString(metricFamilyMetadata.Name)
					result.WriteString("{")
					for m := 0; uint64(m) < labelLen; m++ {
						result.WriteString(metricFamilyMetadata.LabelMap[labels[m]] + "=\"")
						result.WriteString(labelValues[m])
						result.WriteString("\"")
						result.WriteString(",")
					}
					result.WriteString("bucket=\"")
					writeFloat(result, bucket)
					result.WriteString("\"")
					result.WriteString("} ")
					writeFloat(result, metricValue)
				}
				result.WriteString(metricFamilyMetadata.Name + "_sum")
				if labelLen > 0 {
					result.WriteString("{")
					for m := 0; uint64(m) < labelLen; m++ {
						result.WriteString(metricFamilyMetadata.LabelMap[labels[m]] + "=\"")
						result.WriteString(labelValues[m])
						result.WriteString("\"")
						if uint64(m) < labelLen-1 {
							result.WriteString(",")
						}
					}
					result.WriteString("}")
				}
				result.WriteString(" ")
				sumValue, err := reader.readFloat()
				if err != nil {
					return nil, err
				}
				writeFloat(result, sumValue)
				result.WriteString("\n")
				result.WriteString(metricFamilyMetadata.Name + "_count")
				if labelLen > 0 {
					result.WriteString("{")
					for m := 0; uint64(m) < labelLen; m++ {
						result.WriteString(metricFamilyMetadata.LabelMap[labels[m]] + "=\"")
						result.WriteString(labelValues[m])
						result.WriteString("\"")
						if uint64(m) < labelLen-1 {
							result.WriteString(",")
						}
					}
					result.WriteString("}")
				}
				result.WriteString(" ")
				countValue, err := reader.readLeb128Int()
				if err != nil {
					return nil, err
				}
				writeFloat(result, float64(countValue))
				result.WriteString("\n")
			default:
				return nil, errors.New("unexpected metric type")
			}
		}
	}
	return result.Bytes(), nil
}

// writeFloat is equivalent to fmt.Fprint with a float64 argument but hardcodes
// a few common cases for increased efficiency. For non-hardcoded cases, it uses
// strconv.AppendFloat to avoid allocations, similar to writeInt.
func writeFloat(w *bytes.Buffer, f float64) (int, error) {
	switch {
	case f == 1:
		return 1, w.WriteByte('1')
	case f == 0:
		return 1, w.WriteByte('0')
	case f == -1:
		return w.WriteString("-1")
	case math.IsNaN(f):
		return w.WriteString("NaN")
	case math.IsInf(f, +1):
		return w.WriteString("+Inf")
	case math.IsInf(f, -1):
		return w.WriteString("-Inf")
	default:
		bp := numBufPool.Get().(*[]byte)
		*bp = strconv.AppendFloat((*bp)[:0], f, 'g', -1, 64)
		written, err := w.Write(*bp)
		numBufPool.Put(bp)
		return written, err
	}
}
