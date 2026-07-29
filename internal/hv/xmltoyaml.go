package hv

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// xmlNode is a generic representation of an XML element.
type xmlNode struct {
	Attrs    []xml.Attr
	Children map[string][]*xmlNode
	Content  string
}

// decodeElement parses an XML element and its children recursively.
func decodeElement(dec *xml.Decoder, start xml.StartElement) (*xmlNode, error) {
	node := &xmlNode{Attrs: start.Attr, Children: make(map[string][]*xmlNode)}
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			child, err := decodeElement(dec, t)
			if err != nil {
				return nil, err
			}
			name := t.Name.Local
			node.Children[name] = append(node.Children[name], child)
		case xml.EndElement:
			if t.Name == start.Name {
				return node, nil
			}
		case xml.CharData:
			node.Content += string(t)
		}
	}
}

// nodeToYAML converts an xmlNode to a YAML-compatible interface{}.
// Attributes use @ prefix, text content uses _ key.
func nodeToYAML(node *xmlNode) interface{} {
	hasChildren := len(node.Children) > 0
	hasAttrs := len(node.Attrs) > 0
	content := strings.TrimSpace(node.Content)

	if !hasChildren && !hasAttrs {
		if content == "" {
			return nil
		}
		return content
	}

	result := make(map[string]interface{})

	for _, attr := range node.Attrs {
		result["@"+attr.Name.Local] = attr.Value
	}

	if content != "" {
		result["_"] = content
	}

	for name, children := range node.Children {
		if len(children) == 1 {
			result[name] = nodeToYAML(children[0])
		} else {
			items := make([]interface{}, len(children))
			for i, child := range children {
				items[i] = nodeToYAML(child)
			}
			result[name] = items
		}
	}

	return result
}

// WarnMissingFiles checks a libvirt XML string for <source file="..."/> references
// and prints warnings to stderr for any files that do not exist on disk.
// Returns true if all referenced files exist.
func WarnMissingFiles(xmlStr string) bool {
	re := regexp.MustCompile(`source file="([^"]+)"`)
	matches := re.FindAllStringSubmatch(xmlStr, -1)
	allExist := true
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		path := m[1]
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: referenced file does not exist: %s\n", path)
			allExist = false
		}
	}
	return allExist
}

// xmlToYAML converts an XML string to a YAML string.
// Attributes use @ prefix, text content uses _ key.
// Note: XML processing instructions (<?xml ...?>) are silently discarded by
// the decoder. This is intentional — libvirt regenerates the header when
// the XML is passed back via DefineDomain/DefineNetwork.
func xmlToYAML(xmlStr string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	for {
		tok, err := decoder.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			node, err := decodeElement(decoder, t)
			if err != nil {
				return "", err
			}
			yamlData := map[string]interface{}{
				t.Name.Local: nodeToYAML(node),
			}
			yamlBytes, err := yaml.Marshal(yamlData)
			if err != nil {
				return "", err
			}
			return string(yamlBytes), nil
		}
	}
}

type xmlAttr struct {
	Name  string
	Value string
}

// classifyMap separates attributes, children, and text content from a YAML map.
func classifyMap(m map[string]interface{}) (attrs []xmlAttr, children map[string]interface{}, content string) {
	children = make(map[string]interface{})
	for k, v := range m {
		switch {
		case k == "_":
			if s, ok := v.(string); ok {
				content = s
			}
		case strings.HasPrefix(k, "@"):
			attrs = append(attrs, xmlAttr{
				Name:  k[1:],
				Value: fmt.Sprintf("%v", v),
			})
		default:
			children[k] = v
		}
	}
	return
}

// elementToXML builds an XML element from a name and YAML-derived value.
func elementToXML(name string, value interface{}) ([]byte, error) {
	var buf bytes.Buffer

	switch v := value.(type) {
	case nil:
		fmt.Fprintf(&buf, "<%s/>\n", name)

	case string:
		if v == "" {
			fmt.Fprintf(&buf, "<%s/>\n", name)
		} else {
			fmt.Fprintf(&buf, "<%s>%s</%s>\n", name, xmlEscapeText(v), name)
		}

	case map[string]interface{}:
		attrs, children, content := classifyMap(v)

		if len(children) == 0 && content == "" {
			buf.WriteString("<" + name)
			for _, attr := range attrs {
				fmt.Fprintf(&buf, " %s=\"%s\"", attr.Name, xmlEscapeAttr(attr.Value))
			}
			buf.WriteString("/>\n")
			return buf.Bytes(), nil
		}

		buf.WriteString("<" + name)
		for _, attr := range attrs {
			fmt.Fprintf(&buf, " %s=\"%s\"", attr.Name, xmlEscapeAttr(attr.Value))
		}
		buf.WriteString(">")

		if content != "" {
			buf.WriteString(xmlEscapeText(content))
		}
		for childName, childValue := range children {
			childXML, err := elementToXML(childName, childValue)
			if err != nil {
				return nil, err
			}
			buf.Write(childXML)
		}

		fmt.Fprintf(&buf, "</%s>\n", name)

	case []interface{}:
		var out bytes.Buffer
		for _, item := range v {
			childXML, err := elementToXML(name, item)
			if err != nil {
				return nil, err
			}
			out.Write(childXML)
		}
		return out.Bytes(), nil
	}

	return buf.Bytes(), nil
}

// yamlToXML converts a YAML string to an XML string.
// Attributes use @ prefix, text content uses _ key.
func yamlToXML(yamlStr string) (string, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &data); err != nil {
		return "", fmt.Errorf("parse YAML: %w", err)
	}

	for name, value := range data {
		xmlBytes, err := elementToXML(name, value)
		if err != nil {
			return "", err
		}
		return xml.Header + string(xmlBytes), nil
	}
	return "", fmt.Errorf("empty YAML document")
}

// xmlEscapeText escapes text content for XML element bodies.
func xmlEscapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// xmlEscapeAttr escapes attribute values for XML.
func xmlEscapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// XMLToYAML converts a libvirt XML string to YAML format.
// Attributes use @ prefix, text content uses _ key.
func XMLToYAML(xmlStr string) (string, error) {
	return xmlToYAML(xmlStr)
}

// YAMLToXML converts a YAML string back to libvirt XML.
// Attributes use @ prefix, text content uses _ key.
func YAMLToXML(yamlStr string) (string, error) {
	return yamlToXML(yamlStr)
}

// IsXMLMirroringYAML checks if a YAML byte slice uses the XML-mirroring format
// (has "domain" or "network" as a top-level key with a map value) rather than
// the simplified Swift schema. A bare string value (e.g. "domain: production")
// does not qualify — the value must be a nested map.
func IsXMLMirroringYAML(data []byte) bool {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}
	if v, ok := raw["domain"]; ok {
		if _, isMap := v.(map[string]interface{}); isMap {
			return true
		}
	}
	if v, ok := raw["network"]; ok {
		if _, isMap := v.(map[string]interface{}); isMap {
			return true
		}
	}
	return false
}
