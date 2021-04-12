package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	windowCounter2 "github.com/stiflerGit/simpleinsurance-assessment/pkg/rate/counter"
)

func TestServer(t *testing.T) {
	s := &Server{
		counter:  windowCounter2.Must(time.Second, 10),
		logger:   log.Default(),
		filePath: defaultPersistenceFilePath,
	}

	if err := s.Start(context.TODO()); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(s)
	defer ts.Close()

	tests := []struct {
		want []int
	}{
		{
			want: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	}

	for _, tt := range tests {

		for i := 0; i < len(tt.want); i++ {
			res, err := http.Get(ts.URL)
			if err != nil {
				t.Fatal(err)
			}

			JSON, err := io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			response := Response{}
			if err := json.Unmarshal(JSON, &response); err != nil {
				t.Fatal(err)
			}

			got := int(response.Counter)
			if !reflect.DeepEqual(got, tt.want[i]) {
				t.Errorf("at request %d: want = %d, JSON = %d", i, tt.want[i], got)
			}
		}
	}

}
