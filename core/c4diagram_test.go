package core

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
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
				graph: DiagramGraph{
					Nodes: []*Node{{ID: "0"}},
				},
			},
			want: `@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
footer "generated by diagramastext.dev - %date('yyyy-MM-dd')"
Container(0, "0")
@enduml`,
		},
		{
			name: "graph: custom footer",
			args: args{
				graph: DiagramGraph{
					Nodes: []*Node{{ID: "0"}},
					Footer: `  foobar
"bazqux
quxx"  `,
				},
			},
			want: `@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
footer "foobar\n"bazqux\nquxx"
Container(0, "0")
@enduml`,
		},
		{
			name: "graph: custom footer and title",
			args: args{
				graph: DiagramGraph{
					Title:  "foo",
					Footer: "bar",
					Nodes:  []*Node{{ID: "0"}},
				},
			},
			want: `@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
footer "bar"
title "foo"
Container(0, "0")
@enduml`,
		},
		{
			name: "system0",
			args: args{
				graph: DiagramGraph{
					Nodes: []*Node{{ID: "0", Group: "0"}},
				},
			},
			want: `@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
footer "generated by diagramastext.dev - %date('yyyy-MM-dd')"
System_Boundary(0, "0") {
Container(0, "0")
}
@enduml`,
		},
		{
			name: "three nodes: two in one system, two links",
			args: args{
				graph: DiagramGraph{
					Title: "C4 containers to illustrate a data movement",
					Nodes: []*Node{
						{
							ID:         "0",
							Label:      "producer",
							Technology: "Go",
						},
						{
							ID:         "1",
							Label:      "broker",
							Technology: "Kafka",
							IsQueue:    true,
							Group:      "Platform",
							External:   true,
						},
						{
							ID:         "2",
							Label:      "consumer",
							Technology: "Kotlin",
							Group:      "Platform",
							External:   true,
						},
					},
					Links: []*Link{
						{
							From:       "0",
							To:         "1",
							Direction:  "LR",
							Label:      "Publishes domain events",
							Technology: "TCP/Protobuf",
						},
						{
							From:       "2",
							To:         "1",
							Direction:  "RL",
							Label:      "Consumes domain events",
							Technology: "TCP/Protobuf",
						},
					},
				},
			},
			want: `@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
footer "generated by diagramastext.dev - %date('yyyy-MM-dd')"
title "C4 containers to illustrate a data movement"
Container(0, "producer", "Go")
System_Boundary(Platform, "Platform") {
ContainerQueue_Ext(1, "broker", "Kafka")
Container_Ext(2, "consumer", "Kotlin")
}
Rel_R(0, 1, "Publishes domain events", "TCP/Protobuf")
Rel_L(2, 1, "Consumes domain events", "TCP/Protobuf")
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
		n *Node
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
				n: &Node{
					ID: "foo",
				},
			},
			want: `Container(foo, "foo")`,
		},
		{
			name: "ID only, db",
			args: args{
				n: &Node{
					ID:         "foo",
					IsDatabase: true,
				},
			},
			want: `ContainerDb(foo, "foo")`,
		},
		{
			name: "ID only, queue",
			args: args{
				n: &Node{
					ID:      "foo",
					IsQueue: true,
				},
			},
			want: `ContainerQueue(foo, "foo")`,
		},
		{
			name: "ID only, queue+db",
			args: args{
				n: &Node{
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
				n: &Node{
					ID:       "foo",
					External: true,
				},
			},
			want: `Container_Ext(foo, "foo")`,
		},
		{
			name: "ID only, ext db",
			args: args{
				n: &Node{
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
				n: &Node{
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
			args:    args{n: &Node{}},
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
		l *Link
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
				l: &Link{
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
				l: &Link{
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
				l: &Link{
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
				l: &Link{
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
				l: &Link{
					To: "bar",
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "unhappy path: no 'to'",
			args: args{
				l: &Link{
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

func Test_encode64(t *testing.T) {
	type args struct {
		e []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "len(e)==2",
			args: args{
				e: []byte{0, 0},
			},
			want: "0000",
		},
		{
			name: "len(e)==1",
			args: args{
				e: []byte{0},
			},
			want: "0000",
		},
		{
			name: "len(e)==3",
			args: args{
				e: []byte{0, 0, 0},
			},
			want: "0000",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := encode64(tt.args.e); got != tt.want {
					t.Errorf("encode64() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestNewPlantUMLClient(t *testing.T) {
	type args struct {
		optFns []func(*optionsPlantUMLClient)
	}
	tests := []struct {
		name string
		args args
		want ClientGraphToDiagram
	}{
		{
			name: "default",
			args: args{},
			want: &clientPlantUML{
				options: optionsPlantUMLClient{
					httpClient: &http.Client{
						Timeout: defaultTimeoutPlanUML,
					},
				},
				baseURL: baseURLPlanUML,
			},
		},
		{
			name: "custom http client",
			args: args{
				optFns: []func(*optionsPlantUMLClient){
					WithHTTPClientPlantUML(&http.Client{Timeout: 2 * time.Minute}),
				},
			},
			want: &clientPlantUML{
				options: optionsPlantUMLClient{
					httpClient: &http.Client{
						Timeout: 2 * time.Minute,
					},
				},
				baseURL: baseURLPlanUML,
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := NewPlantUMLClient(tt.args.optFns...); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("NewPlantUMLClient() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

type mockPlantUMLClient struct {
	resp *http.Response
	err  error
}

func (m mockPlantUMLClient) Do(req *http.Request) (resp *http.Response, err error) {
	return m.resp, m.err
}

func Test_clientPlantUML_Do(t *testing.T) {
	type fields struct {
		options optionsPlantUMLClient
		baseURL string
	}
	type args struct {
		v DiagramGraph
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				options: optionsPlantUMLClient{
					httpClient: &mockPlantUMLClient{
						resp: &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(`<svg xmlns="http://www.w3.org/2000/svg" "></svg>`)),
						},
					},
				},
				baseURL: baseURLPlanUML,
			},
			args: args{
				v: DiagramGraph{
					Nodes: []*Node{
						{
							ID: "0",
						},
					},
				},
			},
			want:    []byte(`<svg xmlns="http://www.w3.org/2000/svg" "></svg>`),
			wantErr: false,
		},
		{
			name:   "unhappy path: faulty graph",
			fields: fields{},
			args: args{
				v: DiagramGraph{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unhappy path: http client error",
			fields: fields{
				options: optionsPlantUMLClient{
					httpClient: &mockPlantUMLClient{
						err: errors.New("foobar"),
					},
				},
				baseURL: baseURLPlanUML,
			},
			args: args{
				v: DiagramGraph{
					Nodes: []*Node{
						{
							ID: "0",
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unhappy path: server error",
			fields: fields{
				options: optionsPlantUMLClient{
					httpClient: &mockPlantUMLClient{
						resp: &http.Response{
							StatusCode: http.StatusInternalServerError,
						},
					},
				},
				baseURL: baseURLPlanUML,
			},
			args: args{
				v: DiagramGraph{
					Nodes: []*Node{
						{
							ID: "0",
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				c := &clientPlantUML{
					options: tt.fields.options,
					baseURL: tt.fields.baseURL,
				}
				got, err := c.Do(tt.args.v)
				if (err != nil) != tt.wantErr {
					t.Errorf("Do() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Do() got = %v, want %v", got, tt.want)
				}
			},
		)
	}
}
