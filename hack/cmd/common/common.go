package common

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

func SetNestedField(node *yaml.Node, value interface{}, fields ...string) error {

	for i, n := range node.Content {

		if i > 0 && node.Content[i-1].Value == fields[0] {

			// Base case for scalar nodes
			if len(fields) == 1 && n.Kind == yaml.ScalarNode {
				n.SetString(fmt.Sprintf("%s", value))
				break
			}
			// base case for sequence node
			if len(fields) == 1 && n.Kind == yaml.SequenceNode {

				if v, ok := value.([]interface{}); ok {
					var s yaml.Node

					b, err := yaml.Marshal(v)
					if err != nil {
						return err
					}
					if err := yaml.NewDecoder(bytes.NewBuffer(b)).Decode(&s); err != nil {
						return err
					}

					n.Content = s.Content[0].Content
				}
				break
			}

			// Continue to the next level
			return SetNestedField(n, value, fields[1:]...)
		}

		if node.Kind == yaml.DocumentNode {
			return SetNestedField(n, value, fields...)
		}
	}

	return nil
}
