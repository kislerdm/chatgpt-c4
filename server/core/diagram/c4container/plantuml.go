package c4container

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/kislerdm/diagramastext/server/core/errors"

	"github.com/kislerdm/diagramastext/server/core/diagram"
	"github.com/kislerdm/diagramastext/server/core/diagram/c4container/compression"
)

func renderDiagram(ctx context.Context, httpClient diagram.HTTPClient, v *c4ContainersGraph) ([]byte, error) {
	c4ContainersDSL, err := marshal(v)
	if err != nil {
		return nil, err
	}

	requestRoute, err := plantUMLRequest(c4ContainersDSL)
	if err != nil {
		return nil, err
	}

	return callPlantUML(ctx, httpClient, requestRoute)
}

func callPlantUML(ctx context.Context, httpClient diagram.HTTPClient, route string) ([]byte, error) {
	const baseURL = "https://www.plantuml.com/plantuml/"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"svg/"+route, nil)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		if err == nil {
			return nil, errors.New("the response is not ok, status code: " + strconv.Itoa(resp.StatusCode))
		}
		return nil, errors.New(err.Error())
	}

	defer func() { _ = resp.Body.Close() }()

	return io.ReadAll(resp.Body)
}

func writeStrings(w *bytes.Buffer, s ...string) {
	for _, el := range s {
		_, _ = w.WriteString(el)
	}
}

func marshal(c *c4ContainersGraph) ([]byte, error) {
	if len(c.Containers) == 0 {
		return nil, errors.New("no containers found")
	}

	var o bytes.Buffer
	writeStrings(
		&o,
		`@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml`, "\n",
		dslFooter(c.Footer), dslTitle(c.Title),
	)

	groups := map[string][]string{}
	for _, n := range c.Containers {
		if n.ID == "" {
			return nil, errors.New("container must be identified: 'id' attribute")
		}

		if _, ok := groups[n.System]; !ok {
			groups[n.System] = []string{}
		}
		groups[n.System] = append(groups[n.System], dslContainer(n))
	}

	dslSystems(&o, groups)

	writeStrings(&o, "\n")

	for _, l := range c.Rels {
		if l.From == "" || l.To == "" {
			return nil, errors.New("relation must specify the end nodes: 'from' and 'to' attributes")
		}

		dslRelation(&o, l)
		writeStrings(&o, "\n")
	}

	writeStrings(&o, dslLegend(c.WithLegend), "@enduml")

	return o.Bytes(), nil
}

func dslLegend(withLegend bool) string {
	if withLegend {
		return "SHOW_LEGEND()\n"
	}
	return ""
}

func dslRelation(o *bytes.Buffer, l *rel) {
	writeStrings(o, "Rel")

	if d := relationDirection(l.Direction); d != "" {
		writeStrings(o, "_", d)
	}

	writeStrings(o, "(", l.From, ", ", l.To)

	label := l.Label
	if label == "" {
		label = "Uses"
	}
	writeStrings(o, `, "`, stringCleaner(label), `"`)

	if l.Technology != "" {
		writeStrings(o, `, "`, stringCleaner(l.Technology), `"`)
	}

	writeStrings(o, ")")
}

func relationDirection(s string) string {
	switch s := strings.ToUpper(s); s {
	case "LR":
		return "R"
	case "RL":
		return "L"
	case "TD":
		return "D"
	case "DT":
		return "U"
	default:
		return ""
	}
}

func dslSystems(o *bytes.Buffer, groups map[string][]string) {
	tmp := groups

	if members, ok := tmp[""]; ok {
		writeStrings(o, strings.Join(members, "\n"))
		delete(tmp, "")
	}

	for groupName, members := range tmp {
		description := stringCleaner(groupName)
		id := strings.NewReplacer("\n", "", " ", "").Replace(description)
		writeStrings(
			o, "\nSystem_Boundary(", id, `, "`, description, "\") {\n", strings.Join(members, "\n"), "\n}",
		)
	}
}

func dslContainerType(o *bytes.Buffer, n *container) {
	if n.IsUser {
		writeStrings(o, "Person")
	} else {
		writeStrings(o, "Container")
		if n.IsQueue {
			writeStrings(o, "Queue")
		} else if n.IsDatabase {
			writeStrings(o, "Db")
		}
	}

	if n.IsExternal {
		writeStrings(o, "_Ext")
	}
}

func dslContainer(n *container) string {
	var o bytes.Buffer

	dslContainerType(&o, n)

	writeStrings(&o, "(", n.ID)

	label := n.Label
	if label == "" {
		label = n.ID
	}

	writeStrings(&o, `, "`, stringCleaner(label), `"`)

	if n.Technology != "" {
		writeStrings(&o, `, "`, stringCleaner(n.Technology), `"`)
	}

	if n.Description != "" {
		writeStrings(&o, `, "`, stringCleaner(n.Description), `"`)
	}

	writeStrings(&o, ")")

	return o.String()
}

func dslFooter(footer string) string {
	if footer == "" {
		footer = "generated by diagramastext.dev - %date('yyyy-MM-dd')"
	}
	return `footer "` + stringCleaner(footer) + "\"\n"
}

func dslTitle(title string) string {
	if title == "" {
		return ""
	}
	return `title "` + stringCleaner(title) + "\"\n"
}

// plantUMLRequest converts the diagram as code to the 64Bytes encoded string to query plantuml
//
// Example: the diagram's code
// @startuml
//
//	a -> b
//
// @enduml
//
// will be converted to SoWkIImgAStDuL80WaG5NJk592w7rBmKe100
//
// The resulting string to be used to generate C4 diagram
// - as png: GET www.plantuml.com/plantuml/png/SoWkIImgAStDuL80WaG5NJk592w7rBmKe100
// - as svg: GET www.plantuml.com/plantuml/svg/SoWkIImgAStDuL80WaG5NJk592w7rBmKe100
func plantUMLRequest(v []byte) (string, error) {
	zb, err := compress(v)
	if err != nil {
		return "", err
	}
	return encode64(zb), nil
}

func compress(v []byte) ([]byte, error) {
	var options = compression.DefaultOptions()
	var w bytes.Buffer
	if err := compression.Compress(&options, compression.FORMAT_DEFLATE, v, &w); err != nil {
		return nil, errors.New(err.Error())
	}
	return w.Bytes(), nil
}

// FIXME: replace with encode base64.Encoder (?)
// see: https://github.com/kislerdm/diagramastext/pull/20#discussion_r1098013688
func encode64(e []byte) string {
	var r bytes.Buffer
	for i := 0; i < len(e); i += 3 {
		switch len(e) {
		case i + 2:
			r.Write(append3bytes(e[i], e[i+1], 0))
		case i + 1:
			r.Write(append3bytes(e[i], 0, 0))
		default:
			r.Write(append3bytes(e[i], e[i+1], e[i+2]))
		}
	}
	return r.String()
}

func append3bytes(e, n, t byte) []byte {
	c1 := e >> 2
	c2 := (3&e)<<4 | n>>4
	c3 := (15&n)<<2 | t>>6
	c4 := 63 & t

	var buf bytes.Buffer

	buf.WriteByte(encode6bit(c1 & 63))
	buf.WriteByte(encode6bit(c2 & 63))
	buf.WriteByte(encode6bit(c3 & 63))
	buf.WriteByte(encode6bit(c4 & 63))

	return buf.Bytes()
}

func encode6bit(e byte) byte {
	if e < 10 {
		return 48 + e
	}

	e -= 10
	if e < 26 {
		return 65 + e
	}

	e -= 26
	if e < 26 {
		return 97 + e
	}

	e -= 26
	switch e {
	case 0:
		return '-'
	case 1:
		return '_'
	default:
		return '?'
	}
}

func stringCleaner(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
