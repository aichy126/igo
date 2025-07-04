package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TraceID 追踪ID类型
type TraceID string

// SpanID span ID类型
type SpanID string

// Span 表示一个追踪span
type Span struct {
	TraceID    TraceID           `json:"trace_id"`
	SpanID     SpanID            `json:"span_id"`
	ParentID   SpanID            `json:"parent_id,omitempty"`
	Name       string            `json:"name"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Events     []SpanEvent       `json:"events,omitempty"`
	Status     SpanStatus        `json:"status"`
	Error      error             `json:"error,omitempty"`
	Children   []*Span           `json:"children,omitempty"`
}

// SpanEvent span事件
type SpanEvent struct {
	Name       string            `json:"name"`
	Timestamp  time.Time         `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// SpanStatus span状态
type SpanStatus int

const (
	StatusUnset SpanStatus = iota
	StatusOK
	StatusError
)

// Tracer 追踪器接口
type Tracer interface {
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span)
	Inject(ctx context.Context, carrier interface{}) error
	Extract(ctx context.Context, carrier interface{}) (context.Context, error)
	Flush() error
}

// SpanOption span选项
type SpanOption func(*Span)

// WithParent 设置父span
func WithParent(parent *Span) SpanOption {
	return func(s *Span) {
		if parent != nil {
			s.ParentID = parent.SpanID
			s.TraceID = parent.TraceID
		}
	}
}

// WithAttributes 设置span属性
func WithAttributes(attrs map[string]string) SpanOption {
	return func(s *Span) {
		for k, v := range attrs {
			s.Attributes[k] = v
		}
	}
}

// WithStatus 设置span状态
func WithStatus(status SpanStatus) SpanOption {
	return func(s *Span) {
		s.Status = status
	}
}

// WithError 设置span错误
func WithError(err error) SpanOption {
	return func(s *Span) {
		s.Error = err
		s.Status = StatusError
	}
}

// tracerImpl 追踪器实现
type tracerImpl struct {
	serviceName string
	exporter    Exporter
	sampler     Sampler
}

// NewTracer 创建新的追踪器
func NewTracer(serviceName string, opts ...TracerOption) Tracer {
	t := &tracerImpl{
		serviceName: serviceName,
		exporter:    &NoopExporter{},
		sampler:     &AlwaysSampleSampler{},
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// TracerOption 追踪器选项
type TracerOption func(*tracerImpl)

// WithExporter 设置导出器
func WithExporter(exporter Exporter) TracerOption {
	return func(t *tracerImpl) {
		t.exporter = exporter
	}
}

// WithSampler 设置采样器
func WithSampler(sampler Sampler) TracerOption {
	return func(t *tracerImpl) {
		t.sampler = sampler
	}
}

// StartSpan 开始一个新的span
func (t *tracerImpl) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	span := &Span{
		SpanID:     SpanID(uuid.New().String()),
		Name:       name,
		StartTime:  time.Now(),
		Attributes: make(map[string]string),
		Events:     make([]SpanEvent, 0),
		Status:     StatusUnset,
		Children:   make([]*Span, 0),
	}

	// 应用选项
	for _, opt := range opts {
		opt(span)
	}

	// 如果没有父span，创建新的trace
	if span.TraceID == "" {
		span.TraceID = TraceID(uuid.New().String())
	}

	// 检查采样
	if !t.sampler.ShouldSample(span.TraceID) {
		return ctx, nil
	}

	// 将span添加到context
	ctx = context.WithValue(ctx, spanKey{}, span)

	return ctx, span
}

// Inject 注入追踪信息到载体
func (t *tracerImpl) Inject(ctx context.Context, carrier interface{}) error {
	span := GetSpan(ctx)
	if span == nil {
		return nil
	}

	switch c := carrier.(type) {
	case HTTPHeaderCarrier:
		c.Set("X-Trace-ID", string(span.TraceID))
		c.Set("X-Span-ID", string(span.SpanID))
		return nil
	case *HTTPHeaderCarrierImpl:
		c.Set("X-Trace-ID", string(span.TraceID))
		c.Set("X-Span-ID", string(span.SpanID))
		return nil
	default:
		return fmt.Errorf("unsupported carrier type: %T", carrier)
	}
}

// Extract 从载体提取追踪信息
func (t *tracerImpl) Extract(ctx context.Context, carrier interface{}) (context.Context, error) {
	switch c := carrier.(type) {
	case HTTPHeaderCarrier:
		traceID := c.Get("X-Trace-ID")
		spanID := c.Get("X-Span-ID")

		if traceID == "" {
			return ctx, nil
		}

		span := &Span{
			TraceID:    TraceID(traceID),
			SpanID:     SpanID(spanID),
			StartTime:  time.Now(),
			Attributes: make(map[string]string),
			Events:     make([]SpanEvent, 0),
			Status:     StatusUnset,
			Children:   make([]*Span, 0),
		}

		return context.WithValue(ctx, spanKey{}, span), nil
	case *HTTPHeaderCarrierImpl:
		traceID := c.Get("X-Trace-ID")
		spanID := c.Get("X-Span-ID")

		if traceID == "" {
			return ctx, nil
		}

		span := &Span{
			TraceID:    TraceID(traceID),
			SpanID:     SpanID(spanID),
			StartTime:  time.Now(),
			Attributes: make(map[string]string),
			Events:     make([]SpanEvent, 0),
			Status:     StatusUnset,
			Children:   make([]*Span, 0),
		}

		return context.WithValue(ctx, spanKey{}, span), nil
	default:
		return ctx, fmt.Errorf("unsupported carrier type: %T", carrier)
	}
}

// Flush 刷新追踪数据
func (t *tracerImpl) Flush() error {
	return t.exporter.Flush()
}

// spanKey context key类型
type spanKey struct{}

// GetSpan 从context获取span
func GetSpan(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanKey{}).(*Span); ok {
		return span
	}
	return nil
}

// GetTraceID 从context获取trace ID
func GetTraceID(ctx context.Context) TraceID {
	if span := GetSpan(ctx); span != nil {
		return span.TraceID
	}
	return ""
}

// EndSpan 结束span
func EndSpan(span *Span, err error) {
	if span == nil {
		return
	}

	span.EndTime = time.Now()

	if err != nil {
		span.Error = err
		span.Status = StatusError
	} else if span.Status == StatusUnset {
		span.Status = StatusOK
	}
}

// AddEvent 添加span事件
func AddEvent(span *Span, name string, attrs map[string]string) {
	if span == nil {
		return
	}

	event := SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	}

	span.Events = append(span.Events, event)
}

// AddAttribute 添加span属性
func AddAttribute(span *Span, key, value string) {
	if span == nil {
		return
	}

	if span.Attributes == nil {
		span.Attributes = make(map[string]string)
	}

	span.Attributes[key] = value
}

// HTTPHeaderCarrier HTTP header载体接口
type HTTPHeaderCarrier interface {
	Get(key string) string
	Set(key, value string)
}

// Exporter 导出器接口
type Exporter interface {
	Export(spans []*Span) error
	Flush() error
}

// NoopExporter 空操作导出器
type NoopExporter struct{}

func (e *NoopExporter) Export(spans []*Span) error {
	return nil
}

func (e *NoopExporter) Flush() error {
	return nil
}

// ConsoleExporter 控制台导出器
type ConsoleExporter struct {
	logger *zap.Logger
}

func NewConsoleExporter(logger *zap.Logger) *ConsoleExporter {
	return &ConsoleExporter{logger: logger}
}

func (e *ConsoleExporter) Export(spans []*Span) error {
	for _, span := range spans {
		e.logger.Info("Span",
			zap.String("trace_id", string(span.TraceID)),
			zap.String("span_id", string(span.SpanID)),
			zap.String("name", span.Name),
			zap.Time("start_time", span.StartTime),
			zap.Time("end_time", span.EndTime),
			zap.Any("attributes", span.Attributes),
			zap.Any("events", span.Events),
			zap.Int("status", int(span.Status)),
		)
	}
	return nil
}

func (e *ConsoleExporter) Flush() error {
	return nil
}

// Sampler 采样器接口
type Sampler interface {
	ShouldSample(traceID TraceID) bool
}

// AlwaysSampleSampler 总是采样
type AlwaysSampleSampler struct{}

func (s *AlwaysSampleSampler) ShouldSample(traceID TraceID) bool {
	return true
}

// ProbabilitySampler 概率采样
type ProbabilitySampler struct {
	probability float64
}

func NewProbabilitySampler(probability float64) *ProbabilitySampler {
	return &ProbabilitySampler{probability: probability}
}

func (s *ProbabilitySampler) ShouldSample(traceID TraceID) bool {
	// 简单的哈希采样实现
	hash := 0
	for _, char := range traceID {
		hash = 31*hash + int(char)
	}
	return float64(hash%100) < s.probability*100
}

// GlobalTracer 全局追踪器实例
var GlobalTracer Tracer = NewTracer("igo-service")
