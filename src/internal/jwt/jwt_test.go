package jwt_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/edge-easee-adapter/internal/jwt"
)

func TestExpirationDate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		token   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "successful extraction",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJlODYwOTliNi02MTE1LTRmZmEtOWU3My01ODM3MWQ4ODUwMTUiLCJ0eXBlIjoiaWQiLCJpYXQiOjE2ODYwNDExMDUsImV4cCI6MTY4NjA0MTcwNX0.hrP1cHyyOV7I3PM4TMY_0Q2UYokyIugPtFx5HhZbrYk",
			want:  time.Date(2023, time.June, 6, 8, 55, 5, 0, time.UTC),
		},
		{
			name:    "token without expiration date",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJlODYwOTliNi02MTE1LTRmZmEtOWU3My01ODM3MWQ4ODUwMTUiLCJ0eXBlIjoiaWQiLCJpYXQiOjE2ODYwNDExMDV9.uaef87bFclazEKRvu9_-MVsG3T7uoy3U5YPm0kigby0",
			wantErr: true,
		},
		{
			name:    "invalid token",
			token:   "not.even.jwt",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := jwt.ExpirationDate(tc.token)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.want, got)
		})
	}
}
