package message

type Type byte

const (
	// TypCommand ...
	TypeKey Type = '1'

	// Resize ...
	TypeResize Type = '2'

	// Auth ...
	TypeAuth Type = '3'

	// Close ...
	TypeClose Type = '4'

	// Initialize ...
	TypeInitialize Type = '5'
)

func (m *Message) Type() Type {
	return Type(m.msg[0])
}

func (m *Message) SetType(t Type) {
	m.typ = t
}