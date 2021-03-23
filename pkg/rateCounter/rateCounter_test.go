package rateCounter

import (
	"context"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNewWindowCounter(t *testing.T) {
	type args struct {
		duration time.Duration
		opts     []Option
	}
	tests := []struct {
		name    string
		args    args
		want    *RateCounter
		wantErr bool
	}{
		{
			name: "nominal",
			args: args{
				duration: time.Second,
				opts:     nil,
			},
			want: &RateCounter{
				duration:    time.Second,
				counter:     0,
				prevCounter: 0,
				resolution:  defaultResolution,
				counters:    nil,
				head:        0,
				tail:        0,
			},
			wantErr: false,
		},
		{
			name: "with resolution",
			args: args{
				duration: time.Second,
				opts:     []Option{WithResolution(10)},
			},
			want: &RateCounter{
				duration:    time.Second,
				counter:     0,
				prevCounter: 0,
				resolution:  10,
				counters:    nil,
				head:        0,
				tail:        0,
			},
			wantErr: false,
		},
		{
			name: "negative duration",
			args: args{
				duration: -time.Second,
				opts:     nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "period really small",
			args: args{
				duration: time.Nanosecond,
				opts:     nil,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewWindowCounter(tt.args.duration, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWindowCounter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWindowCounter() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRateCounter_Counter(t *testing.T) {
	type fields struct {
		duration    time.Duration
		counter     int64
		prevCounter int64
		resolution  uint64
		counters    []int64
		head        int
		tail        int
	}
	type args struct {
		testDuration      time.Duration
		requestsPerSecond int
		nRoutine          int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int64
	}{
		{
			name: "cold-start one routine",
			fields: fields{
				duration:    time.Second,
				counter:     0,
				prevCounter: 0,
				resolution:  10,
				counters:    nil,
				head:        0,
				tail:        0,
			},
			args: args{
				testDuration:      3 * time.Second,
				requestsPerSecond: 10,
				nRoutine:          1,
			},
			want: 10,
		},
		{
			name: "cold-start two routines",
			fields: fields{
				duration:    time.Second,
				counter:     0,
				prevCounter: 0,
				resolution:  10,
				counters:    nil,
				head:        0,
				tail:        0,
			},
			args: args{
				testDuration:      3 * time.Second,
				nRoutine:          2,
				requestsPerSecond: 10,
			},
			want: 20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RateCounter{
				duration:    tt.fields.duration,
				counter:     tt.fields.counter,
				prevCounter: tt.fields.prevCounter,
				resolution:  tt.fields.resolution,
				counters:    tt.fields.counters,
				head:        tt.fields.head,
				tail:        tt.fields.tail,
			}

			ctx, cancelFunc := context.WithTimeout(context.TODO(), tt.args.testDuration)

			c.Start(ctx)

			wg := sync.WaitGroup{}
			wg.Add(tt.args.nRoutine)
			for i := 0; i < tt.args.nRoutine; i++ {
				go func() {
					defer wg.Done()

					period := computePeriod(time.Second, uint64(tt.args.requestsPerSecond))
					ticks := time.Tick(period)

					for {
						select {
						case <-ctx.Done():
							return
						case <-ticks:
							c.Increase()
						}
					}
				}()
			}

			wg.Wait()
			cancelFunc()

			got := c.Counter()

			// we cannot control the exact time when a routine call increase
			// so a little error is reasonable
			relativeError := math.Abs(float64(got-tt.want) / float64(tt.want))
			if relativeError > 0.1 {
				t.Errorf("TestRateCounter_Counter() relative error greater than 1%%, got = %v, want = %v", got, tt.want)
			}

		})
	}
}

func TestRateCounter_Increase(t *testing.T) {
	type fields struct {
		duration    time.Duration
		counter     int64
		prevCounter int64
		resolution  uint64
		counters    []int64
		head        int
		tail        int
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "nominal",
			fields: fields{
				duration:    time.Second,
				counter:     0,
				prevCounter: 0,
				resolution:  10,
				counters:    nil,
				head:        0,
				tail:        0,
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RateCounter{
				duration:    tt.fields.duration,
				counter:     tt.fields.counter,
				prevCounter: tt.fields.prevCounter,
				resolution:  tt.fields.resolution,
				counters:    tt.fields.counters,
				head:        tt.fields.head,
				tail:        tt.fields.tail,
			}
			if got := c.Increase(); got != tt.want {
				t.Errorf("Increase() = %v, want %v", got, tt.want)
			}
		})
	}
}
