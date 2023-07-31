package message

import "encoding/json"

type Message struct {
	msg []byte

	typ Type

	//
	key    Key
	resize *Resize
	auth   *Auth
	output Output
}

func (m *Message) data() []byte {
	return m.msg[1:]
}

func (m *Message) Msg() []byte {
	return m.msg
}

func (m *Message) Serialize() error {
	switch m.typ {
	case TypeKey:
		m.msg = append([]byte{byte(m.typ)}, m.key...)
	case TypeResize:
		resize, err := json.Marshal(m.resize)
		if err != nil {
			return err
		}
		m.msg = append([]byte{byte(m.typ)}, resize...)
	case TypeAuth:
		auth, err := json.Marshal(m.auth)
		if err != nil {
			return err
		}
		m.msg = append([]byte{byte(m.typ)}, auth...)
	case TypeOutput:
		m.msg = append([]byte{byte(m.typ)}, m.output...)
	}

	return nil
}

func Deserialize(rawMsg []byte) (msg *Message, err error) {
	msg = &Message{msg: rawMsg}
	switch msg.Type() {
	case TypeKey:
		msg.key = msg.data()
	case TypeResize:
		resize := &Resize{}
		err = json.Unmarshal(msg.data(), resize)
		if err != nil {
			return
		}
		msg.resize = resize
	case TypeAuth:
		auth := &Auth{}
		err = json.Unmarshal(msg.data(), auth)
		if err != nil {
			return
		}
		msg.auth = auth
	case TypeOutput:
		msg.output = msg.data()
	}

	return
}
