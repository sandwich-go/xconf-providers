package xvault

import (
	"testing"
)

func Test_getPathAndKey(t *testing.T) {
	type args struct {
		fullPath string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{"normal yaml", args{"pmt/dev/ops_conf.yaml"}, "pmt/dev", "ops_conf.yaml", false},
		{"normal json", args{"pmt/dev/ops_conf.json"}, "pmt/dev", "ops_conf.json", false},
		{"normal", args{"pmt/dev/ops_conf"}, "pmt/dev", "ops_conf", false},
		{"normal", args{"/pmt/dev/ops_conf"}, "/pmt/dev", "ops_conf", false},
		{"normal", args{""}, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := getPathAndKey(tt.args.fullPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPathAndKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getPathAndKey() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getPathAndKey() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
