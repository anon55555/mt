package mt

type NodeMeta struct {
	//mt:len32
	Fields []NodeMetaField

	Inv Inv
}

type NodeMetaField struct {
	Field
	Private bool
}

func (nm *NodeMeta) Field(name string) *NodeMetaField {
	if nm == nil {
		return nil
	}

	for i, f := range nm.Fields {
		if f.Name == name {
			return &nm.Fields[i]
		}
	}

	return nil
}
