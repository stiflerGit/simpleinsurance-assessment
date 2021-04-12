package counter

import (
	"context"
	"math"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	type args struct {
		duration   time.Duration
		resolution uint64
		opts       []Option
	}
	tests := []struct {
		name    string
		args    args
		want    *Counter
		wantErr bool
	}{
		{
			name: "nominal",
			args: args{
				duration:   time.Second,
				resolution: 1000,
				opts:       nil,
			},
			want: &Counter{
				windowDuration: time.Second,
				counter:        0,
				prevCounter:    0,
				resolution:     1000,
				counters:       nil,
				head:           0,
				tail:           0,
			},
			wantErr: false,
		},
		{
			name: "negative windowDuration",
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
			got, err := New(tt.args.duration, tt.args.resolution, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFromFile(t *testing.T) {

	invalidJsonFile, err := os.Create("invalid.json")
	if err != nil {
		t.Fatal(err)
	}
	defer invalidJsonFile.Close()
	defer os.Remove("invalid.json")

	if _, err = invalidJsonFile.Write([]byte("invalidJSON")); err != nil {
		t.Fatal(err)
	}

	type args struct {
		filePath string
		options  []Option
	}
	tests := []struct {
		name    string
		args    args
		wantWc  *Counter
		wantErr bool
	}{
		{
			name: "invalidJSON",
			args: args{
				filePath: invalidJsonFile.Name(),
				options:  nil,
			},
			wantWc:  nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWc, err := NewFromFile(tt.args.filePath, tt.args.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotWc, tt.wantWc) {
				t.Errorf("NewFromFile() gotWc = %v, want %v", gotWc, tt.wantWc)
			}
		})
	}
}

func TestWindowCounter(t *testing.T) {
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
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
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
			want:    10,
			wantErr: false,
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
			want:    20,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Counter{
				windowDuration: tt.fields.duration,
				counter:        tt.fields.counter,
				prevCounter:    tt.fields.prevCounter,
				resolution:     tt.fields.resolution,
				counters:       tt.fields.counters,
				head:           tt.fields.head,
				tail:           tt.fields.tail,
			}

			ctx, cancelFunc := context.WithTimeout(context.TODO(), tt.args.testDuration)

			wg := sync.WaitGroup{}
			wg.Add(1)

			var err error
			go func() {
				defer wg.Done()
				err = c.Run(ctx)
			}()

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

			if (err != nil) != tt.wantErr {
				t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got := c.Value()

			// we cannot control the exact moment when a routine call increase,
			// so a small error is reasonable
			relativeError := math.Abs(float64(got-tt.want) / float64(tt.want))
			if relativeError > 0.1 {
				t.Errorf("TestRateCounter_Counter() relative error greater than 1%%, got = %v, want = %v", got, tt.want)
			}

		})
	}
}
