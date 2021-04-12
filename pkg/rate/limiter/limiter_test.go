package limiter

import (
	"context"
	"testing"
	"time"
)

func makeBoolSlice(size int, value bool) []bool {
	s := make([]bool, size)

	if !value {
		return s
	}

	for i := range s {
		s[i] = true
	}

	return s
}

// TODO: improve tests
func TestLimiter_Allow(t *testing.T) {
	type fields struct {
		duration time.Duration
		limit    int64
	}
	type args struct {
		nReq int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []bool
	}{
		{
			name: "nominal",
			fields: fields{
				duration: time.Second,
				limit:    10,
			},
			args: args{
				nReq: 20,
			},
			want: append(makeBoolSlice(10, true), makeBoolSlice(10, false)...),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Must(tt.fields.duration, tt.fields.limit)
			l.Start(context.TODO())
			for i := 0; i < tt.args.nReq; i++ {
				want := tt.want[i]
				if got := l.IsAllowed(); got != want {
					t.Errorf("request %d, IsAllowed() = %v, want %v", i, got, want)
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}
