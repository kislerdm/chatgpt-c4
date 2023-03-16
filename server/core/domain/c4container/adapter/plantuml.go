package adapter

import (
	"bytes"
	"errors"
	"strings"

	"github.com/kislerdm/diagramastext/server/core/domain/c4container/adapter/compression"
	"github.com/kislerdm/diagramastext/server/core/domain/c4container/port"
)

func C4ContainersGraphPlantUMLRequestMapper(v *port.C4ContainersGraph) (string, error) {
	c4ContainersDSL, err := marshal(v)
	if err != nil {
		return "", err
	}
	return plantUMLRequest(c4ContainersDSL)
}

func writeStrings(w bytes.Buffer, s ...string) error {
	for _, el := range s {
		if _, err := w.WriteString(el); err != nil {
			return err
		}
	}
	return nil
}

func marshal(c *port.C4ContainersGraph) ([]byte, error) {
	if len(c.Containers) == 0 {
		return nil, errors.New("no containers found")
	}

	var o bytes.Buffer

	if err := writeStrings(
		o,
		`@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml`,
		"\n", dslFooter(c.Footer),
		"\n", dslTitle(c.Title),
	); err != nil {
		return nil, err
	}

	groups := map[string][]string{}
	for _, n := range c.Containers {
		containerStr, err := dslContainer(n)
		if err != nil {
			return nil, err
		}

		if _, ok := groups[n.System]; !ok {
			groups[n.System] = []string{}
		}
		groups[n.System] = append(groups[n.System], containerStr)
	}

	if len(groups) > 0 {
		if err := writeStrings(o, dslSystems(o, groups)); err != nil {
			return nil, err
		}
	}

	for _, l := range c.Rels {
		if err := dslRelation(o, l); err != nil {
			return nil, err
		}
		if err := writeStrings(o, "\n"); err != nil {
			return nil, err
		}
	}

	return o.Bytes(), nil
}

func dslRelation(o bytes.Buffer, l *port.Rel) error {
	panic("todo")
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

func dslSystems(o bytes.Buffer, groups map[string][]string) string {
	panic("todo")
}

func dslContainerType(o bytes.Buffer, n *port.Container) error {
	tag := "Container"
	if n.IsUser {
		tag = "User"
	}
	if err := writeStrings(o, tag); err != nil {
		return err
	}

	switch n.IsQueue && n.IsDatabase {
	case true:
	case false:
		if n.IsQueue {
			if err := writeStrings(o, "Queue"); err != nil {
				return err
			}
		}

		if n.IsDatabase {
			if err := writeStrings(o, "Db"); err != nil {
				return err
			}
		}
	}

	if n.IsExternal {
		if err := writeStrings(o, "_Ext"); err != nil {
			return err
		}
	}

	return nil
}

func dslContainer(n *port.Container) (string, error) {
	if n.ID == "" {
		return "", errors.New("container must be identified: 'id' attribute")
	}

	var o bytes.Buffer

	// container type
	if err := dslContainerType(o, n); err != nil {
		return "", err
	}

	// container definition
	if err := writeStrings(o, "("); err != nil {
		return "", err
	}

	if err := writeStrings(o, n.ID); err != nil {
		return "", err
	}

	label := n.Label
	if label == "" {
		label = n.ID
	}

	if err := writeStrings(o, `, "`, stringCleaner(label), `"`); err != nil {
		return "", err
	}

	if n.Technology != "" {
		if err := writeStrings(o, `, "`, stringCleaner(n.Technology), `"`); err != nil {
			return "", err
		}
	}

	if err := writeStrings(o, ")"); err != nil {
		return "", err
	}

	return o.String(), nil
}

func dslFooter(footer string) string {
	if footer == "" {
		return `footer "generated by diagramastext.dev - %date('yyyy-MM-dd')"`
	}
	return `footer "` + stringCleaner(footer) + `"`
}

func dslTitle(title string) string {
	if title == "" {
		return ""
	}
	return `title "` + stringCleaner(title) + `"`
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
		return nil, err
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