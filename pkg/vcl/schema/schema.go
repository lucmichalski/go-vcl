package schema

// File is the root of the data structure
type File struct {
	Body Body
}

// Body contains multiple attributes and blocks
type Body interface{}

type Attributes map[string]*Attribute
type Blocks []*Block

type Block struct {
	Type   string
	Labels []string
	Body   Body
}

// BodySchema represents the desired structure of a body.
type BodySchema struct {
	Attributes []AttributeSchema
	Blocks     []BlockHeaderSchema
}

type Attribute struct {
	Name  string
	Value interface{}
}

type BodyContent struct {
	Attributes Attributes
	Blocks     Blocks
}

// AttributeSchema ...
type AttributeSchema struct {
	Name     string
	Required bool
}

// BlockHeaderSchema ...
type BlockHeaderSchema struct {
	Type       string
	LabelNames []string
	Body       Body
}

// ByType transforms the receiving block sequence into a map from type
// name to block sequences of only that type.
func (bs Blocks) ByType() map[string]Blocks {
	ret := make(map[string]Blocks)
	for _, b := range bs {
		ty := b.Type
		if ret[ty] == nil {
			ret[ty] = make(Blocks, 0, 1)
		}

		ret[ty] = append(ret[ty], b)
	}

	return ret
}
