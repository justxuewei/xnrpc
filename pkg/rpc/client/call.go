package client

type Call struct {
	Seq           uint64
	ServiceMethod string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
}

// notification if call is done(support async call)
func (c *Call) done() {
	c.Done <- c
}
