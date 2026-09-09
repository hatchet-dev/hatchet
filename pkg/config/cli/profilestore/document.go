package profilestore

// The profile file is mutated through yaml.Node surgery rather than a
// marshal/unmarshal round trip so that comments, key ordering, and entries the
// store does not know about (for example metadata written by an embedded
// engine registering itself) survive every write.
//
// Two rules here are load-bearing for interop with the CLI's reader:
//
//   - time.Time values must serialize as UNQUOTED yaml timestamps. The CLI
//     loads the file into a struct with time.Time fields via viper, and a
//     quoted timestamp fails that unmarshal, which makes every CLI command
//     fatal on startup.
//   - The mappings we touch are forced to block style. An empty
//     "profiles: {}" left behind by removing the last profile parses as flow
//     style, and merging a new entry into a flow mapping would make the
//     encoder quote the timestamps.
//
// Keys are matched case-insensitively (the read path is case-insensitive) and
// created lowercased, the file's canonical form.

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

// document is a parsed profiles file whose root node is a mapping.
type document struct {
	doc  *yaml.Node
	root *yaml.Node
}

// loadDocument parses the profiles file at path, creating an empty document
// when the file is absent or empty. Parse failures are returned rather than
// overwriting the file.
func loadDocument(path string) (*document, error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		return &document{
			doc:  &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}},
			root: root,
		}, nil
	}

	doc := &yaml.Node{}
	if err := yaml.Unmarshal(data, doc); err != nil {
		return nil, fmt.Errorf("could not parse %s (leaving it untouched): %w", path, err)
	}

	if len(doc.Content) == 0 {
		root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		return &document{
			doc:  &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}},
			root: root,
		}, nil
	}

	root := doc.Content[0]
	if root.Kind == yaml.ScalarNode && root.Tag == "!!null" {
		*root = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		return &document{doc: doc, root: root}, nil
	}
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s has an unexpected structure (top level is not a mapping); leaving it untouched", path)
	}

	return &document{doc: doc, root: root}, nil
}

// write writes the document to path via a same-directory temp file and rename,
// so concurrent readers never see a partial file. The file is written 0600
// (it contains tokens).
func (d *document) write(path string) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(4)
	if err := enc.Encode(d.doc); err != nil {
		_ = enc.Close()
		return fmt.Errorf("could not encode the profiles file: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("could not encode the profiles file: %w", err)
	}

	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if err := os.WriteFile(tmp, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("could not write %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("could not replace %s: %w", path, err)
	}

	return nil
}

// yamlNodeFor round-trips v through the yaml encoder to obtain its node
// representation, so values like time.Time serialize exactly as the encoder
// would write them (unquoted timestamps).
func yamlNodeFor(v any) (*yaml.Node, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}

	doc := &yaml.Node{}
	if err := yaml.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("could not build a yaml node")
	}

	return doc.Content[0], nil
}

// mappingValue returns the value node for key in mapping m, matching keys
// case-insensitively (viper treats profile keys case-insensitively).
func mappingValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(m.Content); i += 2 {
		if strings.EqualFold(m.Content[i].Value, key) {
			return m.Content[i+1]
		}
	}

	return nil
}

// mappingValueOrCreate returns the mapping value for key, creating an empty
// mapping entry when the key is absent or null. Created keys use key verbatim;
// callers pass keys already lowercased.
func mappingValueOrCreate(m *yaml.Node, key string) (*yaml.Node, error) {
	value := mappingValue(m, key)
	if value == nil {
		value = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		m.Content = append(m.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			value,
		)
		return value, nil
	}

	if value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
		*value = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		return value, nil
	}
	if value.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("the %q key in the profiles file is not a mapping; leaving the file untouched", key)
	}

	return value, nil
}

// setMappingValue replaces the value for key in mapping m (case-insensitive),
// appending the pair when the key is absent. Existing key nodes are kept, so
// comments attached to them survive.
func setMappingValue(m *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if strings.EqualFold(m.Content[i].Value, key) {
			m.Content[i+1] = value
			return
		}
	}

	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

// deleteMappingKey removes key (case-insensitive) and its value from mapping m.
func deleteMappingKey(m *yaml.Node, key string) bool {
	if m == nil || m.Kind != yaml.MappingNode {
		return false
	}

	for i := 0; i+1 < len(m.Content); i += 2 {
		if strings.EqualFold(m.Content[i].Value, key) {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return true
		}
	}

	return false
}
