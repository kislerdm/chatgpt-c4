package core

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_diagramGraph2plantUMLCode(t *testing.T) {
	type args struct {
		graph DiagramGraph
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "default graph",
			args: args{
				graph: DiagramGraph{},
			},
			want: `@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
footer "generated by diagramastext.dev - %date('yyyy-MM-dd')"
@enduml`,
		},
		{
			name: "graph: custom footer",
			args: args{
				graph: DiagramGraph{
					Footer: `  foobar
"bazqux
quxx"  `,
				},
			},
			want: `@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
footer "foobar\n"bazqux\nquxx"
@enduml`,
		},
		{
			name: "graph: custom footer and title",
			args: args{
				graph: DiagramGraph{
					Title:  "foo",
					Footer: "bar",
				},
			},
			want: `@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
footer "bar"
title "foo"
@enduml`,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := diagramGraph2plantUMLCode(tt.args.graph)
				if (err != nil) != tt.wantErr {
					t.Errorf("diagramGraph2plantUMLCode() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("diagramGraph2plantUMLCode() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Test_responseC4Diagram_ToJSON(t *testing.T) {
	mustMarshal := func(v string) []byte {
		o, _ := json.Marshal(
			map[string]string{
				"svg": v,
			},
		)
		return o
	}

	type fields struct {
		SVG string
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "happy path",
			fields: fields{
				SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"></svg>`,
			},
			want: mustMarshal(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"></svg>`),
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				r := ResponseC4Diagram{
					SVG: tt.fields.SVG,
				}
				if got := r.ToJSON(); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ToJSON() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Test_linkDirection(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Rel_R",
			args: args{
				s: "LR",
			},
			want: "R",
		},
		{
			name: "Rel_L",
			args: args{
				s: "RL",
			},
			want: "L",
		},
		{
			name: "Rel_U",
			args: args{
				s: "TD",
			},
			want: "D",
		},
		{
			name: "Rel_D",
			args: args{
				s: "DT",
			},
			want: "U",
		},
		{
			name: "Rel:nothing provided",
			args: args{
				s: "",
			},
			want: "",
		},
		{
			name: "Rel:unknown option provided",
			args: args{
				s: "foobar",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := linkDirection(tt.args.s); got != tt.want {
					t.Errorf("linkDirection() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Test_diagramNode2UML(t *testing.T) {
	type args struct {
		n Node
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "ID only",
			args: args{
				n: Node{
					ID: "foo",
				},
			},
			want: `Container(foo, "foo")`,
		},
		{
			name: "ID only, db",
			args: args{
				n: Node{
					ID:         "foo",
					IsDatabase: true,
				},
			},
			want: `ContainerDb(foo, "foo")`,
		},
		{
			name: "ID only, queue",
			args: args{
				n: Node{
					ID:      "foo",
					IsQueue: true,
				},
			},
			want: `ContainerQueue(foo, "foo")`,
		},
		{
			name: "ID only, queue+db",
			args: args{
				n: Node{
					ID:         "foo",
					IsQueue:    true,
					IsDatabase: true,
				},
			},
			want: `Container(foo, "foo")`,
		},
		{
			name: "ID only, ext",
			args: args{
				n: Node{
					ID:       "foo",
					External: true,
				},
			},
			want: `Container_Ext(foo, "foo")`,
		},
		{
			name: "ID only, ext db",
			args: args{
				n: Node{
					ID:         "foo",
					External:   true,
					IsDatabase: true,
				},
			},
			want: `ContainerDb_Ext(foo, "foo")`,
		},
		{
			name: "core logic example",
			args: args{
				n: Node{
					ID:    "0",
					Label: "Core Logic",
					Technology: `"Go Application"
foobar"`,
				},
			},
			want: `Container(0, "Core Logic", "Go Application"\nfoobar")`,
		},
		{
			name:    "unhappy path",
			args:    args{n: Node{}},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := diagramNode2UML(tt.args.n)
				if (err != nil) != tt.wantErr {
					t.Errorf("diagramNode2UML() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("diagramNode2UML() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Test_diagramLink2UML(t *testing.T) {
	type args struct {
		l Link
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "simple link",
			args: args{
				l: Link{
					From: "foo",
					To:   "bar",
				},
			},
			want:    "Rel(foo, bar)",
			wantErr: false,
		},
		{
			name: "link w. label",
			args: args{
				l: Link{
					From:  "foo",
					To:    "bar",
					Label: "baz",
				},
			},
			want:    `Rel(foo, bar, "baz")`,
			wantErr: false,
		},
		{
			name: "link w. label and technology",
			args: args{
				l: Link{
					From:       "foo",
					To:         "bar",
					Label:      "baz",
					Technology: "HTTP/JSON",
				},
			},
			want:    `Rel(foo, bar, "baz", "HTTP/JSON")`,
			wantErr: false,
		},
		{
			name: "link w. label and technology, direction: on the right",
			args: args{
				l: Link{
					From:       "foo",
					To:         "bar",
					Label:      "baz",
					Technology: "HTTP/JSON",
					Direction:  "LR",
				},
			},
			want:    `Rel_R(foo, bar, "baz", "HTTP/JSON")`,
			wantErr: false,
		},
		{
			name: "unhappy path: no 'from'",
			args: args{
				l: Link{
					To: "bar",
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "unhappy path: no 'to'",
			args: args{
				l: Link{
					From: "foo",
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := diagramLink2UML(tt.args.l)
				if (err != nil) != tt.wantErr {
					t.Errorf("diagramLink2UML() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("diagramLink2UML() got = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestCode2Path(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: `@startuml
    a -> b
@enduml`,
			args: args{
				s: `@startuml
    a -> b
@enduml`,
			},
			want:    "SoWkIImgAStDuL80WaG5NJk592w7rBmKe100",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got, _ := code2Path(tt.args.s); got != tt.want {
					t.Errorf("code2Path() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Test_compress(t *testing.T) {
	type args struct {
		v []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "foo",
			args: args{
				v: []byte("foo"),
			},
			want:    []byte{75, 203, 207, 7, 0},
			wantErr: false,
		},
		{
			name: "foobar",
			args: args{
				v: []byte("foobar"),
			},
			want:    []byte{75, 203, 207, 79, 74, 44, 2, 0},
			wantErr: false,
		},
		{
			name: "@startuml",
			args: args{
				v: []byte(`@startuml`),
			},
			want:    []byte{115, 40, 46, 73, 44, 42, 41, 205, 205, 1, 0},
			wantErr: false,
		},
		{
			name: `foo
bar`,
			args: args{
				v: []byte(`foo
bar`),
			},
			want:    []byte{75, 203, 207, 231, 74, 74, 44, 2, 0},
			wantErr: false,
		},
		{
			name: "->",
			args: args{
				v: []byte(`->`),
			},
			want:    []byte{211, 181, 3, 0},
			wantErr: false,
		},
		{
			name: "a->b",
			args: args{
				v: []byte("a->b"),
			},
			want:    []byte{75, 212, 181, 75, 2, 0},
			wantErr: false,
		},
		{
			name: "a -> b",
			args: args{
				v: []byte("a -> b"),
			},
			want:    []byte{75, 84, 208, 181, 83, 72, 2, 0},
			wantErr: false,
		},
		{
			name: `@startuml
    a -> b
@enduml`,
			args: args{
				v: []byte(`@startuml
    a -> b
@enduml`),
			},
			want: []byte{
				115, 40, 46, 73, 44, 42, 41, 205, 205, 225, 82, 0, 130, 68, 5, 93, 59, 133, 36, 46, 135, 212, 188, 20,
				160, 16, 0,
			},
			wantErr: false,
		},
		{
			name: "unhappy path",
			args: args{
				v: []byte{0},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := compress(tt.args.v)
				if (err != nil) != tt.wantErr {
					t.Errorf("compress() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("compress() got = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Test_encode6bit(t *testing.T) {
	type args struct {
		min, max, threshold byte
	}
	tests := []struct {
		name string
		args args
		want func(in byte, got byte) bool
	}{
		{
			name: "<10bite",
			args: args{0, 10, 48},
		},
		{
			name: "<36bite",
			args: args{10, 36, 65},
		},
		{
			name: "<62bite",
			args: args{36, 62, 97},
		},
		{
			name: "'-'",
			args: args{62, 62, '-'},
		},
		{
			name: "'_'",
			args: args{63, 63, '_'},
		},
		{
			name: "'?'",
			args: args{64, 64, '?'},
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				for i := tt.args.min; i < tt.args.max; i++ {
					got := encode6bit(i)
					want := tt.args.threshold + i - tt.args.min
					if got != want {
						t.Errorf("encode6bit(%v) unexpected result. got = %v, want = %v", i, got, want)
						return
					}
				}
			},
		)
	}

	// syntax signs

}
