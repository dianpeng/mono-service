package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// A customized yaml loader to support customize builtin flags during yaml
// loading, ie supporting environment variables, include, shell include etc ...

var tagResolvers = make(map[string]func(*yaml.Node) (*yaml.Node, error))

func addResolver(tag string, fn func(*yaml.Node) (*yaml.Node, error)) {
	tagResolvers[tag] = fn
}

type Fragment struct {
	content *yaml.Node
}

func (f *Fragment) UnmarshalYAML(value *yaml.Node) error {
	var err error
	// process includes in fragments
	f.content, err = resolveTags(value)
	return err
}

type CustomTagProcessor struct {
	target interface{}
}

func (i *CustomTagProcessor) UnmarshalYAML(value *yaml.Node) error {
	resolved, err := resolveTags(value)
	if err != nil {
		return err
	}
	return resolved.Decode(i.target)
}

func resolveTags(node *yaml.Node) (*yaml.Node, error) {
	for tag, fn := range tagResolvers {
		if node.Tag == tag {
			return fn(node)
		}
	}
	if node.Kind == yaml.SequenceNode || node.Kind == yaml.MappingNode {
		var err error
		for i := range node.Content {
			node.Content[i], err = resolveTags(node.Content[i])
			if err != nil {
				return nil, err
			}
		}
	}
	return node, nil
}

// -----------------------------------------------------------------------------
// builtin resolvers
func resolveInclude(node *yaml.Node) (*yaml.Node, error) {
	if node.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("!inc on a non-scalar node")
	}
	data, err := os.ReadFile(node.Value)
	if err != nil {
		return nil, err
	}
	var f Fragment
	err = yaml.Unmarshal(data, &f)
	return f.content, err
}

func resolveIncludeString(node *yaml.Node) (*yaml.Node, error) {
	if node.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("!inc_string on a non-scalar node")
	}
	data, err := os.ReadFile(node.Value)
	if err != nil {
		return nil, err
	}
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!inc_string",
		Value: string(data),

		Anchor:      node.Anchor,
		Alias:       node.Alias,
		Content:     node.Content,
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}, nil
}

func resolveEnv(node *yaml.Node) (*yaml.Node, error) {
	if node.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("!env on a non-scalar node")
	}
	value := os.Getenv(node.Value)
	return &yaml.Node{
		Kind:        yaml.ScalarNode,
		Tag:         "!inc_string",
		Value:       value,
		Anchor:      node.Anchor,
		Alias:       node.Alias,
		Content:     node.Content,
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}, nil
}

func init() {
	addResolver("!inc", resolveInclude)
	addResolver("!inc_string", resolveIncludeString)
	addResolver("!env", resolveEnv)
}

func loadYaml(data string, output interface{}) error {
	return yaml.Unmarshal(
		[]byte(data),
		&CustomTagProcessor{
			target: output,
		},
	)
}
