package modhelper

import (
	"testing"
)

func Test_validPathElem(t *testing.T) {
	type args struct {
		pathElem string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "empty path element",
			args:    args{pathElem: ""},
			wantErr: true,
		},
		{name: "path element begins with a dot",
			args:    args{pathElem: ".qwerty"},
			wantErr: true,
		},
		{name: "path element ends with a dot",
			args:    args{pathElem: "qwerty."},
			wantErr: true,
		},
		{name: "path element contains invalid character",
			args:    args{pathElem: "qwerty@"},
			wantErr: true,
		},
		{name: "path element must not be a reserved file name on Windows",
			args:    args{pathElem: "Nul"},
			wantErr: true,
		},
		{name: "path element prefix must not be a reserved file name on Windows",
			args:    args{pathElem: "con.qwerty"},
			wantErr: true,
		},
		{name: "path element prefix must not end with a tilde followed by one or more digits",
			args:    args{pathElem: "qwerty~4.aux"},
			wantErr: true,
		},
		{name: "valid path element",
			args:    args{pathElem: "qwerty"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validPathElem(tt.args.pathElem); (err != nil) != tt.wantErr {
				t.Errorf("validPathElem() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidModulePath(t *testing.T) {
	type args struct {
		modPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "empty module path",
			args:    args{modPath: ""},
			wantErr: true,
		},
		{name: "module path begins with a slash",
			args:    args{modPath: "/qwerty/asdfgh/"},
			wantErr: true,
		},
		{name: "module path ends with a slash",
			args:    args{modPath: "qwerty/asdfgh/"},
			wantErr: true,
		},
		{name: "valid module path",
			args:    args{modPath: "qwerty/asdfgh"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidModulePath(tt.args.modPath); (err != nil) != tt.wantErr {
				t.Errorf("ValidModulePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
